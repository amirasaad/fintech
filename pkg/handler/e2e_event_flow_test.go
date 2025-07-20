package handler_test

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	mocks "github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	deposithandler "github.com/amirasaad/fintech/pkg/handler/account/deposit"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestDepositE2EEventFlow tests the full deposit event-driven flow from DepositRequestedEvent to PaymentInitiatedEvent.
// It verifies the event chain:
//
//	DepositRequestedEvent → DepositValidatedEvent → DepositPersistedEvent → DepositConversionDoneEvent → DepositBusinessValidatedEvent → PaymentInitiatedEvent
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
		mu.Lock()
		emitted = append(emitted, eventType)
		mu.Unlock()
	}

	// Register handlers (real logic)
	logger := slog.Default()
	bus := eventbus.NewSimpleEventBus()
	bus.Register("DepositRequestedEvent", func(ctx context.Context, e domain.Event) error {
		track("DepositRequestedEvent")
		deposithandler.Validation(bus, uow, logger)(ctx, e) //nolint:errcheck
		return nil
	})
	bus.Register("DepositValidatedEvent", func(ctx context.Context, e domain.Event) error {
		track("DepositValidatedEvent")
		deposithandler.Persistence(bus, uow, logger)(ctx, e) //nolint:errcheck
		return nil
	})
	bus.Register("DepositPersistedEvent", func(ctx context.Context, e domain.Event) error {
		track("DepositPersistedEvent")
		// Simulate conversion handler emitting DepositConversionDoneEvent
		conversionDone := events.DepositConversionDoneEvent{
			DepositValidatedEvent: e.(events.DepositPersistedEvent).DepositValidatedEvent,
			TransactionID:         e.(events.DepositPersistedEvent).TransactionID,
		}
		bus.Emit(ctx, conversionDone) //nolint:errcheck
		return nil
	})
	bus.Register("DepositConversionDoneEvent", func(ctx context.Context, e domain.Event) error {
		track("DepositConversionDoneEvent")
		deposithandler.ConversionPersistence(bus, uow, logger)(ctx, e) //nolint:errcheck
		deposithandler.BusinessValidation(bus, logger)(ctx, e)         //nolint:errcheck
		return nil
	})
	bus.Register("DepositBusinessValidatedEvent", func(ctx context.Context, e domain.Event) error {
		track("DepositBusinessValidatedEvent")
		// Simulate payment initiation handler emitting PaymentInitiatedEvent
		paymentInitiated := events.PaymentInitiatedEvent{
			ID:            uuid.New().String(),
			TransactionID: e.(events.DepositBusinessValidatedEvent).TransactionID,
			PaymentID:     "",
			Status:        "initiated",
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: e.(events.DepositBusinessValidatedEvent).CorrelationID,
		}
		bus.Emit(ctx, paymentInitiated) //nolint:errcheck
		return nil
	})
	bus.Register("PaymentInitiatedEvent", func(ctx context.Context, e domain.Event) error {
		track("PaymentInitiatedEvent")
		// Simulate payment persistence handler
		return nil
	})

	// Start the chain
	_ = bus.Emit(ctx, events.DepositRequestedEvent{
		FlowEvent: events.FlowEvent{
			AccountID:     accountID,
			UserID:        userID,
			CorrelationID: uuid.New(),
			FlowType:      "deposit",
		},
		Amount:    amount,
		Timestamp: time.Now(),
	})

	// Wait a moment for all handlers to run (since event bus is sync, this is immediate)
	time.Sleep(10 * time.Millisecond)

	// Assert event chain
	assert.Equal(t, []string{
		"DepositRequestedEvent",
		"DepositValidatedEvent",
		"DepositPersistedEvent",
		"DepositConversionDoneEvent",
		"DepositBusinessValidatedEvent",
		"PaymentInitiatedEvent",
	}, emitted, "event chain should match full deposit flow")
}

// TestWithdrawE2EEventFlow tests the full withdraw event-driven flow from WithdrawRequestedEvent to PaymentInitiatedEvent.
// It verifies the event chain:
//
//	WithdrawRequestedEvent → WithdrawValidatedEvent → WithdrawPersistedEvent → WithdrawConversionDoneEvent → WithdrawBusinessValidatedEvent → PaymentInitiatedEvent
//
// The test simulates each handler and tracks the emitted event sequence for correctness.
func TestWithdrawE2EEventFlow(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()
	amount, _ := money.New(100, currency.USD)

	bus := eventbus.NewSimpleEventBus()

	emitted := make([]string, 0, 10)
	var mu sync.Mutex
	track := func(eventType string) {
		mu.Lock()
		emitted = append(emitted, eventType)
		mu.Unlock()
	}

	bus.Register("WithdrawRequestedEvent", func(ctx context.Context, e domain.Event) error {
		track("WithdrawRequestedEvent")
		// Simulate validation handler
		withdrawValidated := events.WithdrawValidatedEvent{
			WithdrawRequestedEvent: e.(events.WithdrawRequestedEvent),
			TargetCurrency:         amount.Currency().String(),
			Account:                nil,
		}
		bus.Emit(ctx, withdrawValidated) //nolint:errcheck
		return nil
	})
	bus.Register("WithdrawValidatedEvent", func(ctx context.Context, e domain.Event) error {
		track("WithdrawValidatedEvent")
		// Simulate persistence handler
		withdrawPersisted := events.WithdrawPersistedEvent{
			WithdrawValidatedEvent: e.(events.WithdrawValidatedEvent),
			TransactionID:          uuid.New(),
		}
		bus.Emit(ctx, withdrawPersisted) //nolint:errcheck
		return nil
	})
	bus.Register("WithdrawPersistedEvent", func(ctx context.Context, e domain.Event) error {
		track("WithdrawPersistedEvent")
		// Simulate conversion handler
		conversionDone := events.WithdrawConversionDoneEvent{
			WithdrawValidatedEvent: e.(events.WithdrawPersistedEvent).WithdrawValidatedEvent,
		}
		bus.Emit(ctx, conversionDone) //nolint:errcheck
		return nil
	})
	bus.Register("WithdrawConversionDoneEvent", func(ctx context.Context, e domain.Event) error {
		track("WithdrawConversionDoneEvent")
		// Simulate business validation
		businessValidated := events.WithdrawBusinessValidatedEvent{
			WithdrawConversionDoneEvent: e.(events.WithdrawConversionDoneEvent),
		}
		bus.Emit(ctx, businessValidated) //nolint:errcheck
		return nil
	})
	bus.Register("WithdrawBusinessValidatedEvent", func(ctx context.Context, e domain.Event) error {
		track("WithdrawBusinessValidatedEvent")
		// Use TransactionID and CorrelationID from previous event
		wbe := e.(events.WithdrawBusinessValidatedEvent)
		paymentInitiated := events.PaymentInitiatedEvent{
			ID:            uuid.New().String(),
			TransactionID: wbe.TransactionID,
			PaymentID:     "",
			Status:        "initiated",
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: wbe.CorrelationID,
		}
		bus.Emit(ctx, paymentInitiated) //nolint:errcheck
		return nil
	})
	bus.Register("PaymentInitiatedEvent", func(ctx context.Context, e domain.Event) error {
		track("PaymentInitiatedEvent")
		return nil
	})

	bus.Emit(ctx, events.WithdrawRequestedEvent{ //nolint:errcheck
		FlowEvent: events.FlowEvent{
			AccountID:     accountID,
			UserID:        userID,
			CorrelationID: uuid.New(),
			FlowType:      "withdraw",
		},
		ID:        uuid.New(),
		Amount:    amount,
		Timestamp: time.Now(),
	}) //nolint:errcheck
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, []string{
		"WithdrawRequestedEvent",
		"WithdrawValidatedEvent",
		"WithdrawPersistedEvent",
		"WithdrawConversionDoneEvent",
		"WithdrawBusinessValidatedEvent",
		"PaymentInitiatedEvent",
	}, emitted, "event chain should match full withdraw flow")
}

// TestTransferE2EEventFlow tests the full transfer event-driven flow from TransferRequestedEvent to TransferCompletedEvent.
// It verifies the event chain:
//
//	TransferRequestedEvent → TransferValidatedEvent → TransferPersistedEvent → TransferConversionDoneEvent → TransferDomainOpDoneEvent → TransferCompletedEvent
//
// The test simulates each handler and tracks the emitted event sequence for correctness.
func TestTransferE2EEventFlow(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()
	amount, _ := money.New(100, currency.USD)

	bus := eventbus.NewSimpleEventBus()

	emitted := make([]string, 0, 10)
	var mu sync.Mutex
	track := func(eventType string) {
		mu.Lock()
		emitted = append(emitted, eventType)
		mu.Unlock()
	}

	bus.Register("TransferRequestedEvent", func(ctx context.Context, e domain.Event) error {
		track("TransferRequestedEvent")
		transferValidated := events.TransferValidatedEvent{
			TransferRequestedEvent: e.(events.TransferRequestedEvent),
		}
		bus.Emit(ctx, transferValidated) //nolint:errcheck
		return nil
	})
	bus.Register("TransferValidatedEvent", func(ctx context.Context, e domain.Event) error {
		track("TransferValidatedEvent")
		transferPersisted := events.TransferPersistedEvent{
			TransferDomainOpDoneEvent: events.TransferDomainOpDoneEvent{
				TransferValidatedEvent: e.(events.TransferValidatedEvent),
			},
		}
		bus.Emit(ctx, transferPersisted) //nolint:errcheck
		return nil
	})
	bus.Register("TransferPersistedEvent", func(ctx context.Context, e domain.Event) error {
		track("TransferPersistedEvent")
		conversionDone := events.TransferConversionDoneEvent{
			TransferValidatedEvent: e.(events.TransferPersistedEvent).TransferValidatedEvent,
		}
		bus.Emit(ctx, conversionDone) //nolint:errcheck
		return nil
	})
	bus.Register("TransferConversionDoneEvent", func(ctx context.Context, e domain.Event) error {
		track("TransferConversionDoneEvent")
		// Simulate conversion persistence and business validation
		transferDomainOpDone := events.TransferDomainOpDoneEvent{
			TransferValidatedEvent: e.(events.TransferConversionDoneEvent).TransferValidatedEvent,
		}
		bus.Emit(ctx, transferDomainOpDone) //nolint:errcheck
		return nil
	})
	bus.Register("TransferDomainOpDoneEvent", func(ctx context.Context, e domain.Event) error {
		track("TransferDomainOpDoneEvent")
		// Simulate internal transfer completion
		domainOpDone := e.(events.TransferDomainOpDoneEvent)
		completed := events.TransferCompletedEvent{
			TransferDomainOpDoneEvent: domainOpDone,
			TxOutID:                   uuid.New(),
			TxInID:                    uuid.New(),
		}
		bus.Emit(ctx, completed) //nolint:errcheck
		return nil
	})
	bus.Register("TransferCompletedEvent", func(ctx context.Context, e domain.Event) error {
		track("TransferCompletedEvent")
		return nil
	})
	bus.Register("PaymentInitiatedEvent", func(ctx context.Context, e domain.Event) error {
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
		ID:             uuid.New(),
		Amount:         amount,
		Source:         "transfer",
		DestAccountID:  uuid.New(),
		ReceiverUserID: uuid.New(),
	}) //nolint:errcheck
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, []string{
		"TransferRequestedEvent",
		"TransferValidatedEvent",
		"TransferPersistedEvent",
		"TransferConversionDoneEvent",
		"TransferDomainOpDoneEvent",
		"TransferCompletedEvent",
	}, emitted, "event chain should match full transfer flow")
}
