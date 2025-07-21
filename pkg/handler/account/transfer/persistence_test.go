package transfer

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	repoaccount "github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFinalPersistence(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	validAmount, _ := money.New(100, currency.USD)

	baseEvent := events.TransferDomainOpDoneEvent{
		TransferValidatedEvent: events.TransferValidatedEvent{
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
		},
	}

	t.Run("successfully persists and emits completed event", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		txRepo := mocks.NewMockTransactionRepository(t)
		accRepo := mocks.NewMockAccountRepository(t)

		sourceAcc, _ := account.New().WithUserID(baseEvent.UserID).WithBalance(500).WithCurrency(currency.USD).Build()
		destAcc, _ := account.New().WithUserID(baseEvent.ReceiverUserID).WithBalance(100).WithCurrency(currency.USD).Build()

		uow.On("Do", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(repository.UnitOfWork) error)
			fn(uow)
		})
		uow.On("GetRepository", (*transaction.Repository)(nil)).Return(txRepo, nil)
		uow.On("GetRepository", (*repoaccount.Repository)(nil)).Return(accRepo, nil)
		txRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
		txRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		accRepo.On("Get", mock.Anything, baseEvent.AccountID).Return(sourceAcc, nil)
		accRepo.On("Get", mock.Anything, baseEvent.DestAccountID).Return(destAcc, nil)
		accRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		bus.On("Emit", ctx, mock.AnythingOfType("events.TransferCompletedEvent")).Return(nil)

		handler := Persistence(bus, uow, logger)
		err := handler(ctx, baseEvent)

		assert.NoError(t, err)
		uow.AssertExpectations(t)
		txRepo.AssertExpectations(t)
		accRepo.AssertExpectations(t)
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
		uow.AssertExpectations(t)
		bus.AssertExpectations(t)
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
