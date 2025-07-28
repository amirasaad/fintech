package transfer_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	transferhandler "github.com/amirasaad/fintech/pkg/handler/account/transfer"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestInitialPersistence(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Common test data
	userID := uuid.New()
	accountID := uuid.New()
	destAccountID := uuid.New()
	correlationID := uuid.New()
	validAmount, _ := money.New(100, "USD")

	t.Run("successfully persists and emits event", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		txRepo := mocks.NewTransactionRepository(t)
		accRepo := mocks.NewAccountRepository(t)

		// Create a fully populated TransferRequestedEvent
		transactionID := uuid.New()
		requestedEvent := events.NewTransferRequestedEvent(
			userID,
			accountID,
			correlationID,
			events.WithTransferRequestedAmount(validAmount),
			events.WithTransferDestAccountID(destAccountID),
		)
		requestedEvent.ID = transactionID // Set the transaction ID

		// Create a fully populated TransferValidatedEvent
		baseEvent := events.NewTransferValidatedEvent(
			userID,
			accountID,
			correlationID,
			events.WithTransferRequestedEvent(*requestedEvent),
		)
		baseEvent.ID = transactionID // Ensure the ID is set in the validated event as well

		// Set up the transaction create matcher with all expected fields
		txCreateMatcher := mock.MatchedBy(func(tx dto.TransactionCreate) bool {
			return tx.AccountID == accountID &&
				tx.UserID == userID &&
				tx.Currency == currency.USD.String() &&
				tx.Status == "pending" &&
				tx.ID != uuid.Nil
		})

		// Mock the unit of work Do function
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			Return(nil).
			Run(func(args mock.Arguments) {
				fn := args.Get(1).(func(repository.UnitOfWork) error)

				// Set up repository mocks for this execution
				uow.On("GetRepository", mock.MatchedBy(func(repoType interface{}) bool {
					_, isTx := repoType.(*transaction.Repository)
					return isTx
				})).Return(txRepo, nil).Once()

				uow.On("GetRepository", mock.MatchedBy(func(repoType interface{}) bool {
					_, isAcc := repoType.(*account.Repository)
					return isAcc
				})).Return(accRepo, nil).Once()

				// Set up account repository expectations
				accRepo.On("Get", mock.Anything, destAccountID).
					Return(&dto.AccountRead{
						ID:       destAccountID,
						UserID:   userID,
						Balance:  1000,
						Currency: currency.USD.String(),
					}, nil).
					Once()

				// Set up transaction repository expectations
				txRepo.On("Create", mock.Anything, txCreateMatcher).
					Return(nil).
					Once()

				// Execute the function under test
				err := fn(uow)
				require.NoError(t, err, "Unexpected error in Do callback")
			})

		bus.On("Emit", mock.Anything, mock.AnythingOfType("*events.ConversionRequestedEvent")).Return(nil)

		handler := transferhandler.InitialPersistence(bus, uow, logger)
		err := handler(ctx, baseEvent)
		assert.NoError(t, err)
	})

	t.Run("emits failed event for repository error", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		// Create a fully populated TransferRequestedEvent
		transactionID := uuid.New()
		requestedEvent := events.NewTransferRequestedEvent(
			userID,
			accountID,
			correlationID,
			events.WithTransferRequestedAmount(validAmount),
			events.WithTransferDestAccountID(destAccountID),
		)
		requestedEvent.ID = transactionID

		// Create a fully populated TransferValidatedEvent
		validatedEvent := events.NewTransferValidatedEvent(
			userID,
			accountID,
			correlationID,
			events.WithTransferRequestedEvent(*requestedEvent),
		)
		validatedEvent.ID = transactionID

		// Mock the unit of work to return an error
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			Return(errors.New("db error"))

		// Expect a TransferFailedEvent to be emitted
		bus.On("Emit", mock.Anything, mock.MatchedBy(func(e common.Event) bool {
			failedEvent, ok := e.(*events.TransferFailedEvent)
			return ok &&
				failedEvent.UserID == userID &&
				failedEvent.AccountID == accountID &&
				failedEvent.CorrelationID == correlationID
		})).Return(nil).Once()

		handler := transferhandler.InitialPersistence(bus, uow, logger)
		err := handler(ctx, validatedEvent)
		assert.Error(t, err)
		bus.AssertExpectations(t)
	})

	t.Run("returns error when repository fails to get repository", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		// Create a fully valid TransferRequestedEvent with all required fields
		transactionID := uuid.New()
		transferRequestedEvent := events.NewTransferRequestedEvent(
			userID,
			accountID,
			correlationID,
			events.WithTransferRequestedAmount(validAmount),
			events.WithTransferDestAccountID(destAccountID),
			events.WithTransferSource("test-source"),
		)
		transferRequestedEvent.ID = transactionID

		// Create a fully valid TransferValidatedEvent
		validatedEvent := events.NewTransferValidatedEvent(
			userID,
			accountID,
			correlationID,
			events.WithTransferRequestedEvent(*transferRequestedEvent),
		)
		validatedEvent.ID = transactionID

		// Mock the repository to return an error when getting the transaction repository
		dbError := errors.New("database error")
		uow.On("GetRepository", mock.Anything).Return(nil, dbError)

		handler := transferhandler.InitialPersistence(bus, uow, logger)
		err := handler(ctx, validatedEvent)

		// Verify the error is properly propagated
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get repo")
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})
}
