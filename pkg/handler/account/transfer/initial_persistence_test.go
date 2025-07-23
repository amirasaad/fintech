package transfer

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
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestInitialPersistence(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	validAmount, _ := money.New(100, currency.USD)

	baseEvent := events.TransferValidatedEvent{
		TransferRequestedEvent: events.TransferRequestedEvent{
			FlowEvent: events.FlowEvent{
				FlowType:      "transfer",
				UserID:        uuid.New(),
				AccountID:     uuid.New(),
				CorrelationID: uuid.New(),
			},
			ID:             uuid.New(),
			Amount:         validAmount,
			Source:         "transfer",
			DestAccountID:  uuid.New(),
			ReceiverUserID: uuid.New(),
		},
	}

	t.Run("successfully persists and emits event", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		txRepo := mocks.NewTransactionRepository(t)
		accRepo := mocks.NewAccountRepository(t)

		// Mock the unit of work Do function to execute the callback
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			Return(nil).
			Run(func(args mock.Arguments) {
				fn := args.Get(1).(func(repository.UnitOfWork) error)

				// Setup mocks inside the Do callback to match the actual implementation
				uow.On("GetRepository", mock.MatchedBy(func(repoType interface{}) bool {
					_, ok := repoType.(*transaction.Repository)
					return ok
				})).Return(txRepo, nil).Once()

				uow.On("GetRepository", mock.MatchedBy(func(repoType interface{}) bool {
					_, ok := repoType.(*account.Repository)
					return ok
				})).Return(accRepo, nil).Once()

				// Mock the destination account lookup
				accRepo.On("Get", mock.Anything, baseEvent.DestAccountID).
					Return(&dto.AccountRead{
						ID:       baseEvent.DestAccountID,
						UserID:   baseEvent.ReceiverUserID,
						Currency: "USD",
					}, nil).Once()

				// Mock the transaction creation
				txRepo.On("Create", mock.Anything, mock.AnythingOfType("dto.TransactionCreate")).
					Return(nil).
					Run(func(args mock.Arguments) {
						txCreate := args.Get(1).(dto.TransactionCreate)
						assert.Equal(t, baseEvent.AccountID, txCreate.AccountID)
						assert.Equal(t, baseEvent.UserID, txCreate.UserID)
						assert.Equal(t, "pending", txCreate.Status)
						assert.Equal(t, "transfer", txCreate.MoneySource)
						assert.Equal(t, validAmount.Negate().Amount(), txCreate.Amount)
					}).Once()

				// Execute the callback
				fn(uow) //nolint:errcheck
			}).Once()

		// Mock the event bus to expect a ConversionRequestedEvent with specific fields
		bus.On("Emit", mock.Anything, mock.MatchedBy(func(event interface{}) bool {
			convEvent, ok := event.(events.ConversionRequestedEvent)
			if !ok {
				return false
			}
			// Compare the Money values using Equals method
			return convEvent.FlowEvent.CorrelationID == baseEvent.CorrelationID &&
				convEvent.Amount.Equals(baseEvent.Amount) &&
				convEvent.TransactionID == baseEvent.ID
		})).Return(nil).Once()

		handler := InitialPersistence(bus, uow, logger)
		err := handler(ctx, baseEvent)

		assert.NoError(t, err)
	})

	t.Run("discards malformed event", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		malformedEvent := baseEvent
		malformedEvent.AccountID = uuid.Nil

		handler := InitialPersistence(bus, uow, logger)
		err := handler(ctx, malformedEvent)

		assert.NoError(t, err)
	})

	t.Run("handles error from repository", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		dbError := errors.New("database error")
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			Return(dbError).
			Once()

		handler := InitialPersistence(bus, uow, logger)
		err := handler(ctx, baseEvent)

		assert.ErrorIs(t, err, dbError)
		// Verify that no events were emitted on error
		bus.AssertNotCalled(t, "Emit")
	})
}
