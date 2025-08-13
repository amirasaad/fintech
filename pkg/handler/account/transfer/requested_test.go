package transfer_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/handler/account/transfer"
	"github.com/amirasaad/fintech/pkg/money"
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
	// Create a logger that discards output for tests using standard library's log/slog
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	// Common test data
	userID := uuid.New()
	accountID := uuid.New()
	destAccountID := uuid.New()
	correlationID := uuid.New()
	validAmount, _ := money.New(100, "USD")

	t.Run("successfully persists and emits event", func(t *testing.T) {
		bus := mocks.NewBus(t)
		uow := mocks.NewUnitOfWork(t)
		txRepo := mocks.NewTransactionRepository(t)
		accRepo := mocks.NewAccountRepository(t)

		// Create a fully populated TransferRequestedEvent
		transactionID := uuid.New()
		requestedEvent := events.NewTransferRequested(
			userID,
			accountID,
			correlationID,
			events.WithTransferRequestedAmount(validAmount),
			events.WithTransferDestAccountID(destAccountID),
		)
		requestedEvent.ID = transactionID // Set the transaction ID

		// Set up the transaction create matcher with all expected fields
		txCreateMatcher := mock.MatchedBy(func(tx dto.TransactionCreate) bool {
			return tx.AccountID == accountID &&
				tx.UserID == userID &&
				tx.Currency == currency.USD.String() &&
				tx.Status == "pending" &&
				tx.ID != uuid.Nil
		})

		// Mock the unit of work Do function
		uow.On(
			"Do",
			mock.Anything,
			mock.AnythingOfType("func(repository.UnitOfWork) error"),
		).
			Return(nil).
			Run(func(args mock.Arguments) {
				fn := args.Get(1).(func(repository.UnitOfWork) error)

				// Set up repository mocks for this execution
				uow.On(
					"GetRepository",
					mock.MatchedBy(func(repoType any) bool {
						_, isTx := repoType.(*transaction.Repository)
						return isTx
					}),
				).
					Return(txRepo, nil).
					Once()

				uow.On(
					"GetRepository",
					mock.MatchedBy(func(repoType any) bool {
						_, isAcc := repoType.(*account.Repository)
						return isAcc
					}),
				).
					Return(accRepo, nil).
					Once()

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

		bus.On(
			"Emit",
			mock.Anything,
			mock.AnythingOfType("*events.CurrencyConversionRequested"),
		).Return(nil)

		handler := transfer.HandleRequested(bus, uow, logger)
		err := handler(ctx, requestedEvent)
		assert.NoError(t, err)
	})

	t.Run("emits_failed_event_for_repository_error", func(t *testing.T) {
		bus := mocks.NewBus(t)
		uow := mocks.NewUnitOfWork(t)

		// Create a validated event
		requestedEvent := &events.TransferRequested{
			FlowEvent: events.FlowEvent{
				FlowType:      "transfer",
				UserID:        userID,
				AccountID:     accountID,
				CorrelationID: correlationID,
			},
			Amount:        validAmount,
			DestAccountID: destAccountID,
		}

		// Mock the unit of work to return a repository that returns an error
		uow.On(
			"Do",
			mock.Anything,
			mock.AnythingOfType("func(repository.UnitOfWork) error"),
		).
			Run(func(args mock.Arguments) {
				t.Log("Running Do callback...")
				// Get the callback function
				fn := args.Get(1).(func(repository.UnitOfWork) error)

				t.Log("Setting up GetRepository mocks...")
				// Mock the GetRepository method to return a mock transaction repository first
				txRepo := mocks.NewTransactionRepository(t)
				// Then mock the account repository
				accRepo := mocks.NewAccountRepository(t)

				// First call to GetRepository is for transaction repository
				uow.On(
					"GetRepository",
					mock.MatchedBy(func(repoType interface{}) bool {
						_, isTx := repoType.(*transaction.Repository)
						return isTx
					}),
				).
					Return(txRepo, nil).
					Once()

				// Second call to GetRepository is for account repository
				uow.On("GetRepository", mock.MatchedBy(func(repoType interface{}) bool {
					_, isAcc := repoType.(*account.Repository)
					return isAcc
				})).Return(accRepo, nil).Once()

				t.Log("Setting up account repository mock...")
				// Mock the Get method to return an error
				accRepo.On("Get", mock.Anything, destAccountID).
					Return(nil, errors.New("account not found")).
					Run(func(args mock.Arguments) {
						t.Logf("Account repository Get called with: %v", args.Get(1))
					}).
					Once()

				t.Log("Calling the callback function...")
				// Call the callback function with the mock UoW
				err := fn(uow)
				if err != nil {
					t.Logf("Callback returned error: %v", err)
				}
				require.Error(t, err)
			}).
			Return(errors.New("db error"))

		// The handler should return an error and not emit any events
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)

		handler := transfer.HandleRequested(bus, uow, logger)
		// Pass the event as a common.Event interface
		err := handler(ctx, requestedEvent)

		// Verify the error is properly propagated
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")

		// Verify no events were emitted
		bus.AssertExpectations(t)
	})

	t.Run("returns error when repository fails to get repository", func(t *testing.T) {
		bus := mocks.NewBus(t)
		uow := mocks.NewUnitOfWork(t)

		// Create a fully valid TransferValidatedEvent with all required fields
		transactionID := uuid.New()
		requestedEvent := &events.TransferRequested{
			FlowEvent: events.FlowEvent{
				FlowType:      "transfer",
				UserID:        userID,
				AccountID:     accountID,
				CorrelationID: transactionID,
			},
			Amount:        validAmount,
			Source:        "test-source",
			DestAccountID: destAccountID,
		}

		// Mock the unit of work to return an error when getting the repository
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			Run(func(args mock.Arguments) {
				t.Log("Running Do callback...")
				// When Do is called, it should try to get the repository
				t.Log("Setting up GetRepository mocks...")

				// First GetRepository call is for the transaction repository
				txRepo := mocks.NewTransactionRepository(t)
				uow.On("GetRepository", mock.MatchedBy(func(repoType interface{}) bool {
					_, isTx := repoType.(*transaction.Repository)
					return isTx
				})).Return(txRepo, nil).Once()

				// Second GetRepository call is for the account repository
				accRepo := mocks.NewAccountRepository(t)
				uow.On("GetRepository", mock.MatchedBy(func(repoType interface{}) bool {
					_, isAcc := repoType.(*account.Repository)
					return isAcc
				})).Return(accRepo, nil).Once()

				// Mock the account repository to return an error when Get is called
				accRepo.On(
					"Get",
					mock.Anything,
					destAccountID).
					Return(
						nil,
						errors.New("db error"),
					).Once()

				// Execute the callback to simulate the actual flow
				t.Log("Calling the callback function...")
				fn := args.Get(1).(func(repository.UnitOfWork) error)
				err := fn(uow)
				t.Log("Callback returned error:", err)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get destination account: db error")
			}).
			Return(errors.New("db error")) // Return the error from Do

		// The handler should return an error and not emit any events
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)

		handler := transfer.HandleRequested(bus, uow, logger)
		t.Log("Calling handler...")
		err := handler(ctx, requestedEvent)

		// Debug log the actual error
		if err != nil {
			t.Logf("Handler returned error: %v", err)
		}

		// Verify the error is properly propagated
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")

		// Verify no events were emitted
		bus.AssertExpectations(t)

		// Verify all repository expectations were met
		uow.AssertExpectations(t)
	})
}
