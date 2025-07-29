package transfer_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/handler/account/transfer"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
)

// newTestTransferDomainOpDoneEvent creates a properly constructed TransferDomainOpDoneEvent for testing
func newTestTransferDomainOpDoneEvent(userID, accountID, destAccountID, correlationID uuid.UUID, amount money.Money) *events.TransferDomainOpDoneEvent {
	transferRequested := events.NewTransferRequestedEvent(
		userID, accountID, correlationID,
		events.WithTransferRequestedAmount(amount),
		events.WithTransferDestAccountID(destAccountID),
	)

	validated := events.NewTransferValidatedEvent(
		userID, accountID, correlationID,
		events.WithTransferRequestedEvent(*transferRequested),
	)

	conversionDone := events.ConversionDoneEvent{
		FlowEvent: events.FlowEvent{
			FlowType:      "conversion_done",
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: correlationID,
		},
		ConvertedAmount: amount,
	}

	domainOpDone := &events.TransferDomainOpDoneEvent{
		FlowEvent: events.FlowEvent{
			FlowType:      "transfer_domain_op_done",
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: correlationID,
		},
		TransferValidatedEvent: *validated,
		ConversionDoneEvent:    conversionDone,
	}

	return domainOpDone
}

// matchTransferFailedEvent is a helper to match TransferFailedEvent with loose field matching
func matchTransferFailedEvent(t *testing.T, expectedAccountID, expectedDestAccountID uuid.UUID, expectedReason string) interface{} {
	return mock.MatchedBy(func(e interface{}) bool {
		switch evt := e.(type) {
		case events.TransferFailedEvent:
			// Check the important fields with loose matching
			matched := evt.TransferRequestedEvent.AccountID == expectedAccountID &&
				evt.TransferRequestedEvent.DestAccountID == expectedDestAccountID &&
				strings.Contains(evt.Reason, expectedReason)
			if !matched {
				t.Logf("Event fields don't match. Got: %+v, Want AccountID: %v, DestAccountID: %v, Reason contains: %s",
					evt, expectedAccountID, expectedDestAccountID, expectedReason)
			}
			return matched
		default:
			t.Logf("Expected events.TransferFailedEvent, got %T", e)
			return false
		}
	})
}

func TestPersistence(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("successfully persists transaction and emits event", func(t *testing.T) {
		// Setup test data
		amount, _ := money.New(100, currency.USD)
		userID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		destAccountID := uuid.New()

		// Create a properly constructed event using our helper
		event := newTestTransferDomainOpDoneEvent(userID, accountID, destAccountID, correlationID, amount)

		// Setup mocks
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		txRepo := mocks.NewTransactionRepository(t)
		accRepo := mocks.NewAccountRepository(t)

		// Setup mock expectations for successful execution path
		// 1. Expect GetRepository to be called for transaction repository
		uow.On("GetRepository", mock.MatchedBy(func(repoType interface{}) bool {
			_, isTx := repoType.(*transaction.Repository)
			return isTx
		})).Return(txRepo, nil).Once()

		// 2. Expect GetRepository to be called for account repository
		uow.On("GetRepository", mock.MatchedBy(func(repoType interface{}) bool {
			_, isAcc := repoType.(*account.Repository)
			return isAcc
		})).Return(accRepo, nil).Once()

		// 3. Setup transaction repository to update transaction status
		completedStatus := "completed"
		txRepo.On("Update", mock.Anything, event.TransactionID, dto.TransactionUpdate{Status: &completedStatus}).
			Return(nil).Once()

		// 4. Setup account repository to return source and destination accounts
		sourceAcc := &dto.AccountRead{ID: accountID, Balance: 1000.0}
		destAcc := &dto.AccountRead{ID: destAccountID, Balance: 500.0}
		accRepo.On("Get", mock.Anything, accountID).Return(sourceAcc, nil).Once()
		accRepo.On("Get", mock.Anything, destAccountID).Return(destAcc, nil).Once()

		// 5. Setup account repository to update balances
		updatedSourceBalance := 900.0 // 1000 - 100
		updatedDestBalance := 600.0   // 500 + 100
		accRepo.On("Update", mock.Anything, accountID, dto.AccountUpdate{Balance: &updatedSourceBalance}).
			Return(nil).Once()
		accRepo.On("Update", mock.Anything, destAccountID, dto.AccountUpdate{Balance: &updatedDestBalance}).
			Return(nil).Once()

		// 6. Setup UnitOfWork Do to execute the transaction
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			Run(func(args mock.Arguments) {
				// Get the callback function
				fn, ok := args.Get(1).(func(repository.UnitOfWork) error)
				if !ok {
					t.Fatal("failed to cast callback function")
				}
				// Execute the function to simulate the transaction
				_ = fn(uow)
			}).Return(nil).Once()

		// 7. Expect TransferCompletedEvent to be emitted
		bus.On("Emit", mock.Anything, mock.MatchedBy(func(e interface{}) bool {
			evt, ok := e.(*events.TransferCompletedEvent)
			if !ok {
				t.Logf("Expected *events.TransferCompletedEvent, got %T", e)
				return false
			}
			// Check the fields from the embedded TransferRequestedEvent
			req := evt.TransferDomainOpDoneEvent.TransferValidatedEvent.TransferRequestedEvent
			return req.AccountID == accountID &&
				req.DestAccountID == destAccountID &&
				req.Amount.Equals(amount)
		})).Return(nil).Once()

		// Create a test logger that writes to the test output
		var logOutput bytes.Buffer
		testLogger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		// Create and execute the handler
		handler := transfer.Persistence(bus, uow, testLogger)
		err := handler(ctx, event)

		// Assertions
		assert.NoError(t, err)

		// Verify all mock expectations were met
		if !t.Failed() {
			// Only check these if the main test hasn't already failed
			bus.AssertExpectations(t)
			uow.AssertExpectations(t)
			txRepo.AssertExpectations(t)
			accRepo.AssertExpectations(t)
		}

		// Log the output for debugging
		t.Logf("Test logs:\n%s", logOutput.String())

		// Log all mock calls that were made for debugging
		t.Log("Mock calls:")
		for _, call := range uow.Calls {
			t.Logf("UoW call: %s %v", call.Method, call.Arguments)
		}
		for _, call := range txRepo.Calls {
			t.Logf("TxRepo call: %s %v", call.Method, call.Arguments)
		}
		for _, call := range accRepo.Calls {
			t.Logf("AccRepo call: %s %v", call.Method, call.Arguments)
		}
		for _, call := range bus.Calls {
			t.Logf("Bus call: %s %v", call.Method, call.Arguments)
		}
		uow.AssertExpectations(t)
		txRepo.AssertExpectations(t)
		accRepo.AssertExpectations(t)
	})

	t.Run("fails gracefully when repository error occurs", func(t *testing.T) {
		// Setup test data
		amount, _ := money.New(100, currency.USD)
		userID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		destAccountID := uuid.New()

		// Create a properly constructed event using our helper
		event := newTestTransferDomainOpDoneEvent(userID, accountID, destAccountID, correlationID, amount)

		// Setup mocks
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		txRepo := mocks.NewTransactionRepository(t)
		accRepo := mocks.NewAccountRepository(t)

		// Setup mock expectations for repository error case
		// 1. Expect GetRepository to be called for transaction repository
		uow.On("GetRepository", mock.MatchedBy(func(repoType interface{}) bool {
			_, isTx := repoType.(*transaction.Repository)
			return isTx
		})).Return(txRepo, nil).Once()

		// 2. Expect GetRepository to be called for account repository
		uow.On("GetRepository", mock.MatchedBy(func(repoType interface{}) bool {
			_, isAcc := repoType.(*account.Repository)
			return isAcc
		})).Return(accRepo, nil).Once()

		// 3. Setup account repository to return error on Get
		dbError := errors.New("database error")
		accRepo.On("Get", mock.Anything, accountID).Return(nil, dbError).Once()

		// 4. Setup UnitOfWork Do to execute the transaction
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			Run(func(args mock.Arguments) {
				// Get the callback function
				fn, ok := args.Get(1).(func(repository.UnitOfWork) error)
				if !ok {
					t.Fatal("failed to cast callback function")
				}
				// Execute the function to simulate the transaction
				_ = fn(uow)
			}).Return(nil).Once()

		// 5. Expect TransferCompletedEvent to be emitted
		// The handler emits a TransferCompletedEvent even on error, as it handles the error internally
		bus.On("Emit", mock.Anything, mock.MatchedBy(func(e interface{}) bool {
			ev, ok := e.(*events.TransferCompletedEvent)
			if !ok {
				t.Logf("Expected *events.TransferCompletedEvent, got %T", e)
				return false
			}
			// Check the fields from the embedded TransferRequestedEvent
			req := ev.TransferDomainOpDoneEvent.TransferValidatedEvent.TransferRequestedEvent
			return req.AccountID == accountID &&
				req.DestAccountID == destAccountID &&
				req.Amount.Equals(amount)
		})).Return(nil).Once()

		handler := transfer.Persistence(bus, uow, logger)
		err := handler(context.Background(), event)

		// Assertions - The handler should return nil since it emits an event on error
		require.NoError(t, err)

		// Verify all mock expectations were met
		if !t.Failed() {
			bus.AssertExpectations(t)
			uow.AssertExpectations(t)
			txRepo.AssertExpectations(t)
			accRepo.AssertExpectations(t)
		}
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		// Setup test data
		amount, _ := money.New(100, currency.USD)
		userID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		destAccountID := uuid.New()

		// Create a properly constructed event using our helper
		event := newTestTransferDomainOpDoneEvent(userID, accountID, destAccountID, correlationID, amount)

		// Setup mocks
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		// Setup mock expectations for repository failure case
		repoError := errors.New("failed to get repository")

		// Setup UnitOfWork Do to execute the function and return the repository error
		// We'll expect this to be called twice: once for the main transaction and once to mark as failed
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			Run(func(args mock.Arguments) {
				// This simulates what happens inside the Do function
				fn := args.Get(1).(func(repository.UnitOfWork) error)
				// When the function calls GetRepository, make it return an error
				mockUOW := mocks.NewMockUnitOfWork(t)
				mockUOW.On("GetRepository", mock.Anything).Return(nil, repoError).Once()
				// Execute the function which should trigger the error
				_ = fn(mockUOW)
			}).
			Return(repoError).Twice() // Expect this to be called twice

		// Expect TransferFailedEvent to be emitted
		bus.On("Emit", mock.Anything, matchTransferFailedEvent(t, accountID, destAccountID, "failed to get repository")).Return(nil).Once()

		handler := transfer.Persistence(bus, uow, logger)
		err := handler(context.Background(), event)

		// Assertions - The handler should return nil since it emits an event on error
		require.NoError(t, err)

		// Verify all mock expectations were met
		if !t.Failed() {
			bus.AssertExpectations(t)
			uow.AssertExpectations(t)
		}
	})
}
