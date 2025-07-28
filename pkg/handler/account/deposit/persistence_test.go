package deposit_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/handler/account/deposit"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPersistence(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("successfully persists deposit and emits persisted event", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		userID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		// Create a deposit validated event with all required fields
		event := NewValidDepositValidatedEvent(userID, accountID, correlationID, amount)

		// Setup test data
		accRepo := mocks.NewAccountRepository(t)
		accRead := &dto.AccountRead{
			ID:       accountID,
			UserID:   userID,
			Balance:  100000,
			Currency: "USD",
		}

		// Create a transaction repository mock
		txRepo := mocks.NewTransactionRepository(t)

		// Setup the uow.Do to execute the callback function
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			Return(nil).
			Run(func(args mock.Arguments) {
				// This runs when Do is called, simulating the transaction
				cb := args.Get(1).(func(repository.UnitOfWork) error)

				// Setup mocks that will be used inside the callback
				uow.Mock.On("GetRepository", (*account.Repository)(nil)).Return(accRepo, nil).Once()
				uow.Mock.On("GetRepository", (*transaction.Repository)(nil)).Return(txRepo, nil).Once()

				// Mock the account repository Get call
				accRepo.On("Get", mock.Anything, accountID).Return(accRead, nil).Once()

				// Mock the transaction repository Create call
				txRepo.On("Create", mock.Anything, mock.MatchedBy(func(tx dto.TransactionCreate) bool {
					return tx.UserID == userID &&
						tx.AccountID == accountID &&
						tx.Currency == "USD" &&
						tx.Status == "created" &&
						tx.Amount == 10000 // 100 * 100 (minor units)
				})).Return(nil).Once()

				// Execute the callback
				err := cb(uow)
				assert.NoError(t, err, "callback should not return error")
			}).Once()

		// Expect the Emit call for DepositPersistedEvent
		bus.On("Emit", mock.Anything, mock.MatchedBy(func(e interface{}) bool {
			persistedEvent, ok := e.(*events.DepositPersistedEvent)
			if !ok {
				return false
			}
			return persistedEvent.UserID == userID &&
				persistedEvent.AccountID == accountID &&
				persistedEvent.CorrelationID == correlationID &&
				persistedEvent.TransactionID != uuid.Nil &&
				persistedEvent.Amount.Equals(amount)
		})).Return(nil).Once()
		bus.On("Emit", mock.Anything, mock.MatchedBy(func(e interface{}) bool {
			persistedEvent, ok := e.(*events.ConversionRequestedEvent)
			if !ok {
				return false
			}
			return persistedEvent.UserID == userID &&
				persistedEvent.AccountID == accountID &&
				persistedEvent.CorrelationID == correlationID &&
				persistedEvent.TransactionID != uuid.Nil &&
				persistedEvent.Amount.Equals(amount)
		})).Return(nil).Once()

		// Execute
		handler := deposit.Persistence(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err)
		bus.AssertExpectations(t)
	})

	t.Run("handles unexpected event type gracefully", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		// Use a different event type
		event := events.WithdrawValidatedEvent{}

		// Execute
		handler := deposit.Persistence(bus, uow, logger)
		err := handler(ctx, &event)

		// Assert
		assert.NoError(t, err)
		uow.AssertNotCalled(t, "Do", mock.Anything, mock.Anything)
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

	t.Run("handles persistence error", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		userID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		// Create a deposit validated event with all required fields
		event := NewValidDepositValidatedEvent(userID, accountID, correlationID, amount)

		// Mock persistence error
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).Return(errors.New("persistence error")).Once()

		// Execute
		handler := deposit.Persistence(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.Error(t, err)
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

}
