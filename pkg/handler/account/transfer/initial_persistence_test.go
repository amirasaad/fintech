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
	"github.com/amirasaad/fintech/pkg/repository"
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

		uow.On("Do", ctx, mock.AnythingOfType("func(repository.UnitOfWork) error")).Return(
			func(ctx context.Context, fn func(uow repository.UnitOfWork) error) error {
				return fn(uow)
			},
		).Once()
		uow.On("GetRepository", mock.Anything).Return(txRepo, nil).Once()
		txRepo.On("Create", ctx, mock.Anything).Return(nil).Once()
		bus.On("Emit", ctx, mock.AnythingOfType("events.ConversionRequestedEvent")).Return(nil).Once()

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
		uow.AssertNotCalled(t, "Do")
		bus.AssertNotCalled(t, "Emit")
	})

	t.Run("handles error from repository", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		dbError := errors.New("database error")
		uow.On("Do", ctx, mock.AnythingOfType("func(repository.UnitOfWork) error")).Return(dbError).Once()

		handler := InitialPersistence(bus, uow, logger)
		err := handler(ctx, baseEvent)

		assert.ErrorIs(t, err, dbError)
		bus.AssertNotCalled(t, "Emit")
	})
}
