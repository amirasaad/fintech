package withdraw_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/handler/account/withdraw"
	repoaccount "github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBusinessValidation(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	validAmount, _ := money.New(100, "USD")

	baseEvent := events.WithdrawValidatedEvent{
		WithdrawRequestedEvent: events.WithdrawRequestedEvent{
			Amount: validAmount,
		},
	}

	t.Run("successfully validates and emits domain op done event", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewMockAccountRepository(t)

		accDto := &dto.AccountRead{ID: baseEvent.AccountID, UserID: baseEvent.UserID, Balance: 500, Currency: "USD"}

		uow.On("GetRepository", (*repoaccount.Repository)(nil)).Return(accRepo, nil)
		accRepo.On("Get", ctx, baseEvent.AccountID).Return(accDto, nil)
		bus.On("Emit", ctx, mock.AnythingOfType("events.WithdrawDomainOpDoneEvent")).Return(nil)

		handler := withdraw.BusinessValidation(bus, logger)
		err := handler(ctx, baseEvent)

		assert.NoError(t, err)
		uow.AssertExpectations(t)
		accRepo.AssertExpectations(t)
		bus.AssertExpectations(t)
	})

	t.Run("emits failed event for insufficient funds", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewMockAccountRepository(t)

		accDto := &dto.AccountRead{ID: baseEvent.AccountID, UserID: baseEvent.UserID, Balance: 50, Currency: "USD"} // Not enough balance

		uow.On("GetRepository", (*repoaccount.Repository)(nil)).Return(accRepo, nil)
		accRepo.On("Get", ctx, baseEvent.AccountID).Return(accDto, nil)
		bus.On("Emit", ctx, mock.AnythingOfType("events.WithdrawFailedEvent")).Return(nil)

		handler := withdraw.BusinessValidation(bus, logger)
		err := handler(ctx, baseEvent)

		assert.NoError(t, err)
	})

	t.Run("emits failed event when account not found", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewMockAccountRepository(t)

		uow.On("GetRepository", (*repoaccount.Repository)(nil)).Return(accRepo, nil)
		accRepo.On("Get", ctx, baseEvent.AccountID).Return(nil, errors.New("not found"))
		bus.On("Emit", ctx, mock.AnythingOfType("events.WithdrawFailedEvent")).Return(nil)

		handler := withdraw.BusinessValidation(bus, logger)
		err := handler(ctx, baseEvent)

		assert.NoError(t, err)
	})

	t.Run("returns error on repository failure", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		uow.On("GetRepository", (*repoaccount.Repository)(nil)).Return(nil, errors.New("repo error"))
		bus.On("Emit", ctx, mock.AnythingOfType("events.WithdrawFailedEvent")).Return(nil)

		handler := withdraw.BusinessValidation(bus, logger)
		err := handler(ctx, baseEvent)

		assert.Error(t, err)
	})
}
