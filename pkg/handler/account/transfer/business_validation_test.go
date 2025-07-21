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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Helper to create a valid event for tests
func newValidConversionDoneEvent(t *testing.T) events.TransferConversionDoneEvent {
	t.Helper()
	amount, err := money.New(100, currency.USD)
	assert.NoError(t, err)

	return events.TransferConversionDoneEvent{
		TransferValidatedEvent: events.TransferValidatedEvent{
			TransferRequestedEvent: events.TransferRequestedEvent{
				FlowEvent: events.FlowEvent{
					FlowType:  "transfer",
					UserID:    uuid.New(),
					AccountID: uuid.New(),
				},
				Amount: amount,
			},
		},
		ConversionDoneEvent: events.ConversionDoneEvent{
			FlowEvent: events.FlowEvent{
				FlowType:  "transfer",
				UserID:    uuid.New(),
				AccountID: uuid.New(),
			},
			ToAmount: amount,
		},
	}
}

func TestBusinessValidation(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("successfully validates and emits domain op done event", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)
		event := newValidConversionDoneEvent(t)

		uow.On("GetRepository", mock.Anything).Return(accRepo, nil).Once()
		accRepo.On("Get", ctx, event.AccountID).Return(&dto.AccountRead{Balance: 20000, UserID: event.UserID}, nil).Once()
		bus.On("Emit", ctx, mock.AnythingOfType("events.TransferDomainOpDoneEvent")).Return(nil).Once()

		handler := BusinessValidation(bus, uow, logger)
		err := handler(ctx, event)

		assert.NoError(t, err)
	})

	t.Run("emits failed event for insufficient funds", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)
		event := newValidConversionDoneEvent(t)

		uow.On("GetRepository", mock.Anything).Return(accRepo, nil).Once()
		accRepo.On("Get", ctx, event.AccountID).Return(&dto.AccountRead{Balance: 50, UserID: event.UserID}, nil).Once()
		bus.On("Emit", ctx, mock.AnythingOfType("events.TransferFailedEvent")).Return(nil).Once()

		handler := BusinessValidation(bus, uow, logger)
		err := handler(ctx, event)

		assert.NoError(t, err)
	})

	t.Run("emits failed event when account not found", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)
		event := newValidConversionDoneEvent(t)

		uow.On("GetRepository", mock.Anything).Return(accRepo, nil).Once()
		accRepo.On("Get", ctx, event.AccountID).Return(nil, errors.New("not found")).Once()
		bus.On("Emit", ctx, mock.AnythingOfType("events.TransferFailedEvent")).Return(nil).Once()

		handler := BusinessValidation(bus, uow, logger)
		err := handler(ctx, event)

		assert.NoError(t, err)
	})

	t.Run("returns error on repository failure", func(t *testing.T) {
		uow := mocks.NewMockUnitOfWork(t)
		event := newValidConversionDoneEvent(t)
		dbError := errors.New("database error")

		uow.On("GetRepository", mock.Anything).Return(nil, dbError).Once()

		handler := BusinessValidation(mocks.NewMockBus(t), uow, logger)
		err := handler(ctx, event)

		assert.ErrorIs(t, err, dbError)
	})
}
