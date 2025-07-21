package withdraw_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/handler/account/withdraw"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestValidation(t *testing.T) {
	bus := eventbus.NewSimpleEventBus()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	mockUOW := new(mocks.MockUnitOfWork)
	mockAccountRepo := new(mocks.AccountRepository)

	ctx := context.Background()

	t.Run("should emit WithdrawValidatedEvent on valid request", func(t *testing.T) {
		amount, _ := money.New(100, "USD")
		req := events.WithdrawRequestedEvent{
			FlowEvent: events.FlowEvent{
				AccountID: uuid.New(),
			},
			ID:     uuid.New(),
			Amount: amount,
		}

		mockUOW.On("GetRepository", (*account.Repository)(nil)).Return(mockAccountRepo, nil)
		mockAccountRepo.On("Get", ctx, req.AccountID).Return(&dto.AccountRead{Currency: "USD"}, nil)

		handler := withdraw.Validation(bus, mockUOW, logger)
		err := handler(ctx, req)

		assert.NoError(t, err)
		mockUOW.AssertExpectations(t)
		mockAccountRepo.AssertExpectations(t)
	})

	t.Run("should emit WithdrawFailedEvent on invalid request", func(t *testing.T) {
		amount, _ := money.New(0, "USD")
		req := events.WithdrawRequestedEvent{
			FlowEvent: events.FlowEvent{
				AccountID: uuid.Nil,
			},
			ID:     uuid.New(),
			Amount: amount,
		}

		handler := withdraw.Validation(bus, mockUOW, logger)
		err := handler(ctx, req)

		assert.NoError(t, err)
	})
}
