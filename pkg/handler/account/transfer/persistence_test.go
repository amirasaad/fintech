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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFinalPersistence(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	validAmount, _ := money.New(100, currency.USD)

	userID := uuid.New()
	accountID := uuid.New()
	destAccountID := uuid.New()
	receiverUserID := uuid.New()
	correlationID := uuid.New()
	transactionID := uuid.New()

	baseEvent := events.TransferDomainOpDoneEvent{
		TransferValidatedEvent: events.TransferValidatedEvent{
			TransferRequestedEvent: events.TransferRequestedEvent{
				FlowEvent: events.FlowEvent{
					FlowType:      "transfer",
					UserID:        userID,
					AccountID:     accountID,
					CorrelationID: correlationID,
				},
				ID:             transactionID,
				Amount:         validAmount,
				Source:         "transfer",
				DestAccountID:  destAccountID,
				ReceiverUserID: receiverUserID,
			},
		},
		ConversionDoneEvent: events.ConversionDoneEvent{
			FlowEvent: events.FlowEvent{
				FlowType:      "transfer",
				UserID:        userID,
				AccountID:     accountID,
				CorrelationID: correlationID,
			},
			TransactionID:   transactionID,
			ConvertedAmount: validAmount,
		},
		TransactionID: transactionID,
	}

	t.Run("successfully persists and emits completed event", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		// Mock the unit of work Do function to return success
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			Return(nil).Once()

		// Mock the event bus to expect a TransferCompletedEvent
		bus.On("Emit", mock.Anything, mock.AnythingOfType("events.TransferCompletedEvent")).
			Return(nil).
			Run(func(args mock.Arguments) {
				_, event := args.Get(0), args.Get(1)
				completedEvent, ok := event.(events.TransferCompletedEvent)
				assert.True(t, ok, "expected TransferCompletedEvent")
				// Just check that TxInID is set (not nil)
				assert.NotEqual(t, uuid.Nil, completedEvent.TxInID)
			}).Once()

		handler := Persistence(bus, uow, logger)
		err := handler(ctx, baseEvent)

		assert.NoError(t, err)
		uow.AssertExpectations(t)
		bus.AssertExpectations(t)
	})

	t.Run("emits failed event on database error", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		dbError := errors.New("database error")
		uow.On("Do", mock.Anything, mock.Anything).Return(dbError)
		bus.On("Emit", ctx, mock.AnythingOfType("events.TransferFailedEvent")).Return(nil)

		handler := Persistence(bus, uow, logger)
		err := handler(ctx, baseEvent)

		assert.NoError(t, err)
	})

	t.Run("discards malformed event", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		malformedEvent := baseEvent
		malformedEvent.AccountID = uuid.Nil

		handler := Persistence(bus, uow, logger)
		err := handler(ctx, malformedEvent)

		assert.NoError(t, err)
		uow.AssertNotCalled(t, "Do", mock.Anything, mock.Anything)
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})
}
