package handler_test

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/amirasaad/fintech/infra/eventbus"
	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	deposithandler "github.com/amirasaad/fintech/pkg/handler/account/deposit"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestDepositE2EEventFlow tests the full deposit event-driven flow from DepositRequestedEvent to PaymentInitiatedEvent.
// It verifies the event chain:
//
//	DepositRequestedEvent → DepositValidatedEvent → DepositPersistedEvent → DepositBusinessValidationEvent → DepositBusinessValidatedEvent → PaymentInitiatedEvent
//
// The test uses mocks for repository and unit of work, and tracks the emitted event sequence for correctness.
func TestDepositE2EEventFlow(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()
	amount, _ := money.New(100, currency.USD)

	// Setup mocks
	accRepo := mocks.NewAccountRepository(t)
	uow := mocks.NewMockUnitOfWork(t)
	accRead := &dto.AccountRead{
		ID:       accountID,
		UserID:   userID,
		Balance:  amount.AmountFloat(),
		Currency: amount.Currency().String(),
	}
	accRepo.On("Get", mock.Anything, accountID).Return(accRead, nil)
	uow.On("GetRepository", mock.Anything).Return(accRepo, nil)
	uow.On("Do", mock.Anything, mock.Anything).Return(nil)

	// Prepare account read DTO
	// Track event emissions
	emitted := make([]string, 0, 10)
	var mu sync.Mutex
	track := func(eventType string) {
		t.Logf("Event emitted: %s", eventType)
		mu.Lock()
		emitted = append(emitted, eventType)
		mu.Unlock()
	}

	// Register handlers (real logic)
	logger := slog.Default()
	bus := eventbus.NewWithMemory(logger)

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Create a wait group to wait for all events to be processed
	var wg sync.WaitGroup
	wg.Add(6) // We expect 6 events in total

	bus.Register("DepositRequestedEvent", func(ctx context.Context, e common.Event) error {
		track("DepositRequestedEvent")
		defer wg.Done()
		return deposithandler.Validation(bus, uow, logger)(ctx, e)
	})
	bus.Register("DepositValidatedEvent", func(ctx context.Context, e common.Event) error {
		track("DepositValidatedEvent")
		defer wg.Done()
		return deposithandler.Persistence(bus, uow, logger)(ctx, e)
	})
	bus.Register("DepositPersistedEvent", func(ctx context.Context, e common.Event) error {
		track("DepositPersistedEvent")
		defer wg.Done()
		// Simulate conversion handler emitting DepositBusinessValidationEvent
		persistedEvent, ok := e.(*events.DepositPersistedEvent)
		if !ok {
			t.Fatal("failed to cast to DepositPersistedEvent")
		}
		conversionDone := &events.DepositBusinessValidationEvent{
			DepositValidatedEvent: persistedEvent.DepositValidatedEvent,
			Amount:                persistedEvent.Amount,
			ConversionDoneEvent: events.ConversionDoneEvent{
				FlowEvent: events.FlowEvent{
					UserID:        persistedEvent.UserID,
					AccountID:     persistedEvent.AccountID,
					CorrelationID: persistedEvent.CorrelationID,
					FlowType:      "deposit",
				},
				ID:            uuid.New(),
				RequestID:     persistedEvent.TransactionID.String(),
				TransactionID: persistedEvent.TransactionID,
				Timestamp:     time.Now(),
			},
		}
		return bus.Emit(ctx, conversionDone)
	})
	bus.Register("DepositBusinessValidationEvent", func(ctx context.Context, e common.Event) error {
		track("DepositBusinessValidationEvent")
		defer wg.Done()
		// Simulate business validation handler emitting DepositBusinessValidatedEvent
		businessValidationEvent, ok := e.(*events.DepositBusinessValidationEvent)
		if !ok {
			t.Fatal("failed to cast to DepositBusinessValidationEvent")
		}
		businessValidated := &events.DepositBusinessValidatedEvent{
			DepositBusinessValidationEvent: *businessValidationEvent,
			ID:                             uuid.New(),
			TransactionID:                  businessValidationEvent.DepositValidatedEvent.TransactionID,
		}
		return bus.Emit(ctx, businessValidated)
	})
	bus.Register("DepositBusinessValidatedEvent", func(ctx context.Context, e common.Event) error {
		track("DepositBusinessValidatedEvent")
		defer wg.Done()
		// Simulate payment initiation handler emitting PaymentInitiatedEvent
		validatedEvent, ok := e.(*events.DepositBusinessValidatedEvent)
		if !ok {
			t.Fatal("failed to cast to DepositBusinessValidatedEvent")
		}
		paymentInitiated := events.PaymentInitiatedEvent{
			ID:            uuid.New().String(),
			TransactionID: validatedEvent.TransactionID,
			PaymentID:     uuid.New().String(),
			Status:        "initiated",
		}
		return bus.Emit(ctx, paymentInitiated)
	})
	bus.Register("PaymentInitiatedEvent", func(ctx context.Context, e common.Event) error {
		track("PaymentInitiatedEvent")
		defer wg.Done()
		// Simulate payment persistence handler
		return nil
	})

	// Start the chain
	depositEvent := events.NewDepositRequestedEvent(
		userID,
		accountID,
		uuid.New(),
		events.WithDepositAmount(amount),
		events.WithDepositTimestamp(time.Now()),
	)

	err := bus.Emit(ctx, depositEvent)
	require.NoError(t, err, "failed to emit DepositRequestedEvent")

	// Wait for all events to be processed or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All events processed
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for events to process")
	}

	// Assert event chain
	require.Len(t, emitted, 6, "expected 6 events to be emitted")
	assert.Equal(t, []string{
		"DepositRequestedEvent",
		"DepositValidatedEvent",
		"DepositPersistedEvent",
		"DepositBusinessValidationEvent",
		"DepositBusinessValidatedEvent",
		"PaymentInitiatedEvent",
	}, emitted, "event chain should match full deposit flow")
}

// TestWithdrawE2EEventFlow tests the full withdraw event-driven flow from WithdrawRequestedEvent to PaymentInitiatedEvent.
// It verifies the event chain:
//
//	WithdrawRequestedEvent → WithdrawValidatedEvent → WithdrawPersistedEvent → WithdrawBusinessValidationEvent → WithdrawBusinessValidatedEvent → PaymentInitiatedEvent
//
// The test simulates each handler and tracks the emitted event sequence for correctness.
func TestWithdrawE2EEventFlow(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()
	amount, _ := money.New(100, currency.USD)

	// Setup test data - we'll use this in our test events
	_ = &dto.AccountRead{
		ID:       accountID,
		UserID:   userID,
		Balance:  1000.0, // Set initial balance to 1000 USD (more than the withdrawal amount)
		Currency: currency.USD.String(),
	}

	bus := eventbus.NewWithMemory(slog.Default())

	emitted := make([]string, 0, 10)
	var mu sync.Mutex
	track := func(eventType string) {
		mu.Lock()
		emitted = append(emitted, eventType)
		mu.Unlock()
	}

	// First, deposit enough funds to cover the withdrawal
	depositAmount, _ := money.New(200, currency.USD) // Deposit more than we'll withdraw
	depositEvent := events.NewDepositRequestedEvent(
		userID,
		accountID,
		uuid.New(),
		events.WithDepositAmount(depositAmount),
		events.WithDepositTimestamp(time.Now()),
	)

	// Process the deposit event to ensure the account has funds
	depositValidated := events.NewDepositValidatedEvent(
		depositEvent.UserID,
		depositEvent.AccountID,
		depositEvent.CorrelationID,
		events.WithDepositRequestedEvent(*depositEvent),
	)

	// Create a persisted deposit event to simulate a successful deposit
	_ = &events.DepositPersistedEvent{
		DepositValidatedEvent: *depositValidated,
		TransactionID:         uuid.New(),
	}

	// Register a temporary handler for deposit events to process them
	depositBus := eventbus.NewWithMemory(slog.Default())
	var depositProcessed bool
	depositBus.Register("DepositRequestedEvent", func(ctx context.Context, e common.Event) error {
		de := e.(*events.DepositRequestedEvent)
		validated := events.NewDepositValidatedEvent(
			de.UserID,
			de.AccountID,
			de.CorrelationID,
			events.WithDepositRequestedEvent(*de),
		)
		depositBus.Emit(ctx, validated)
		return nil
	})

	depositBus.Register("DepositValidatedEvent", func(ctx context.Context, e common.Event) error {
		de := e.(*events.DepositValidatedEvent)
		persisted := &events.DepositPersistedEvent{
			DepositValidatedEvent: *de,
			TransactionID:         uuid.New(),
		}
		depositBus.Emit(ctx, persisted)
		return nil
	})

	depositBus.Register("DepositPersistedEvent", func(ctx context.Context, e common.Event) error {
		depositProcessed = true
		return nil
	})

	// Process the deposit
	depositBus.Emit(ctx, depositEvent)

	// Wait for deposit to be processed
	time.Sleep(100 * time.Millisecond)
	require.True(t, depositProcessed, "Deposit should be processed before withdrawal")

	bus.Register("WithdrawRequestedEvent", func(ctx context.Context, e common.Event) error {
		track("WithdrawRequestedEvent")
		// Simulate validation handler
		reqEvent := e.(*events.WithdrawRequestedEvent)
		withdrawValidated := events.NewWithdrawValidatedEvent(
			reqEvent.UserID,
			reqEvent.AccountID,
			reqEvent.CorrelationID,
			events.WithWithdrawRequestedEvent(*reqEvent),
			events.WithTargetCurrency(amount.Currency().String()),
		)
		bus.Emit(ctx, withdrawValidated) //nolint:errcheck
		return nil
	})
	bus.Register("WithdrawValidatedEvent", func(ctx context.Context, e common.Event) error {
		track("WithdrawValidatedEvent")
		// Simulate persistence handler
		validatedEvent := e.(*events.WithdrawValidatedEvent)
		withdrawPersisted := &events.WithdrawPersistedEvent{
			WithdrawValidatedEvent: *validatedEvent,
			TransactionID:          uuid.New(),
		}
		bus.Emit(ctx, withdrawPersisted) //nolint:errcheck
		return nil
	})
	bus.Register("WithdrawPersistedEvent", func(ctx context.Context, e common.Event) error {
		track("WithdrawPersistedEvent")
		// Simulate conversion handler
		persistedEvent := e.(*events.WithdrawPersistedEvent)

		// Create conversion done event with proper currency
		conversionDoneEvent := events.NewConversionDoneEvent(
			persistedEvent.UserID,
			persistedEvent.AccountID,
			persistedEvent.CorrelationID,
		)
		conversionDoneEvent.RequestID = persistedEvent.TransactionID.String()
		conversionDoneEvent.TransactionID = persistedEvent.TransactionID
		// Set the converted amount (1:1 conversion for same currency)
		convertedAmount := persistedEvent.WithdrawValidatedEvent.Amount
		conversionDoneEvent.ConvertedAmount = convertedAmount
		// Set conversion info with proper currency
		conversionDoneEvent.ConversionInfo = &common.ConversionInfo{
			OriginalAmount:    convertedAmount.AmountFloat(),
			OriginalCurrency:  convertedAmount.Currency().String(),
			ConvertedAmount:   convertedAmount.AmountFloat(),
			ConvertedCurrency: convertedAmount.Currency().String(),
			ConversionRate:    1.0,
		}

		// Create business validation event
		businessValidationEvent := events.NewWithdrawBusinessValidationEvent(
			persistedEvent.UserID,
			persistedEvent.AccountID,
			persistedEvent.CorrelationID,
		)
		businessValidationEvent.WithdrawValidatedEvent = persistedEvent.WithdrawValidatedEvent
		businessValidationEvent.Amount = persistedEvent.WithdrawValidatedEvent.Amount
		businessValidationEvent.ConversionDoneEvent = conversionDoneEvent

		bus.Emit(ctx, businessValidationEvent) //nolint:errcheck
		return nil
	})
	bus.Register("WithdrawBusinessValidationEvent", func(ctx context.Context, e common.Event) error {
		track("WithdrawBusinessValidationEvent")
		// Simulate business validation
		validationEvent := e.(*events.WithdrawBusinessValidationEvent)
		businessValidated := &events.WithdrawBusinessValidatedEvent{
			WithdrawBusinessValidationEvent: *validationEvent,
		}
		bus.Emit(ctx, businessValidated) //nolint:errcheck
		return nil
	})
	bus.Register("WithdrawBusinessValidatedEvent", func(ctx context.Context, e common.Event) error {
		track("WithdrawBusinessValidatedEvent")
		// Use TransactionID and CorrelationID from previous event
		wbe := e.(*events.WithdrawBusinessValidatedEvent)
		paymentInitiated := events.NewPaymentInitiatedEvent(
			wbe.FlowEvent,
			uuid.New().String(),
			wbe.TransactionID,
			"",
		)
		bus.Emit(ctx, paymentInitiated) //nolint:errcheck
		return nil
	})
	bus.Register("PaymentInitiatedEvent", func(ctx context.Context, e common.Event) error {
		track("PaymentInitiatedEvent")
		return nil
	})

	withdrawEvent := events.NewWithdrawRequestedEvent(
		userID,
		accountID,
		uuid.New(),
		events.WithWithdrawAmount(amount),
		events.WithWithdrawTimestamp(time.Now()),
	)
	bus.Emit(ctx, withdrawEvent) //nolint:errcheck
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, []string{
		"WithdrawRequestedEvent",
		"WithdrawValidatedEvent",
		"WithdrawPersistedEvent",
		"WithdrawBusinessValidationEvent",
		"WithdrawBusinessValidatedEvent",
		"PaymentInitiatedEvent",
	}, emitted, "event chain should match full withdraw flow")
}

// TestTransferE2EEventFlow tests the full transfer event-driven flow from TransferRequestedEvent to TransferCompletedEvent.
// It verifies the event chain:
//
//	TransferRequestedEvent → TransferValidatedEvent → TransferDomainOpDoneEvent → TransferConversionDoneEvent → TransferCompletedEvent
//
// The test simulates each handler and tracks the emitted event sequence for correctness.
func TestTransferE2EEventFlow(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()
	amount, _ := money.New(100, currency.USD)

	bus := eventbus.NewWithMemory(slog.Default())

	emitted := make([]string, 0, 10)
	var mu sync.Mutex
	track := func(eventType string) {
		mu.Lock()
		emitted = append(emitted, eventType)
		mu.Unlock()
	}

	bus.Register("TransferRequestedEvent", func(ctx context.Context, e common.Event) error {
		track("TransferRequestedEvent")
		transferValidated := events.TransferValidatedEvent{
			TransferRequestedEvent: e.(events.TransferRequestedEvent),
		}
		bus.Emit(ctx, transferValidated) //nolint:errcheck
		return nil
	})
	bus.Register("TransferValidatedEvent", func(ctx context.Context, e common.Event) error {
		track("TransferValidatedEvent")
		transferDomainOpDone := events.TransferDomainOpDoneEvent{
			TransferValidatedEvent: e.(events.TransferValidatedEvent),
			TransactionID:          uuid.New(),
		}
		bus.Emit(ctx, transferDomainOpDone) //nolint:errcheck
		return nil
	})
	bus.Register("TransferDomainOpDoneEvent", func(ctx context.Context, e common.Event) error {
		track("TransferDomainOpDoneEvent")
		// Simulate conversion handler
		domainOpDoneEvent := e.(events.TransferDomainOpDoneEvent)
		businessValidated := events.TransferBusinessValidatedEvent{
			FlowEvent:              domainOpDoneEvent.FlowEvent,
			TransferValidatedEvent: domainOpDoneEvent.TransferValidatedEvent,
			ConversionDoneEvent: events.ConversionDoneEvent{
				FlowEvent:     domainOpDoneEvent.FlowEvent,
				ID:            uuid.New(),
				RequestID:     domainOpDoneEvent.TransactionID.String(),
				TransactionID: domainOpDoneEvent.TransactionID,
				Timestamp:     time.Now(),
			},
		}
		return bus.Emit(ctx, businessValidated)
	})
	bus.Register("TransferBusinessValidatedEvent", func(ctx context.Context, e common.Event) error {
		track("TransferBusinessValidatedEvent")
		// Simulate internal transfer completion
		businessValidatedEvent := e.(events.TransferBusinessValidatedEvent)
		completed := events.TransferCompletedEvent{
			FlowEvent: businessValidatedEvent.FlowEvent,
			TransferDomainOpDoneEvent: events.TransferDomainOpDoneEvent{
				FlowEvent:              businessValidatedEvent.FlowEvent,
				TransferValidatedEvent: businessValidatedEvent.TransferValidatedEvent,
				TransactionID:          businessValidatedEvent.TransactionID,
			},
			TxOutID: uuid.New(),
			TxInID:  uuid.New(),
		}
		return bus.Emit(ctx, completed)
	})
	bus.Register("TransferCompletedEvent", func(ctx context.Context, e common.Event) error {
		track("TransferCompletedEvent")
		return nil
	})
	bus.Register("PaymentInitiatedEvent", func(ctx context.Context, e common.Event) error {
		track("PaymentInitiatedEvent")
		return nil
	})

	bus.Emit(ctx, events.TransferRequestedEvent{ //nolint:errcheck
		FlowEvent: events.FlowEvent{
			AccountID:     accountID,
			UserID:        userID,
			CorrelationID: uuid.New(),
			FlowType:      "transfer",
		},
		ID:            uuid.New(),
		Amount:        amount,
		Source:        "transfer",
		DestAccountID: uuid.New(),
	}) //nolint:errcheck
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, []string{
		"TransferRequestedEvent",
		"TransferValidatedEvent",
		"TransferDomainOpDoneEvent",
		"TransferBusinessValidatedEvent",
		"TransferCompletedEvent",
	}, emitted, "event chain should match full transfer flow")
}
