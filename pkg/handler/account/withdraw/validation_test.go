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
	"github.com/amirasaad/fintech/pkg/handler/account/withdraw"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestValidation(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx := context.Background()

	t.Run("should emit WithdrawValidatedEvent on valid request", func(t *testing.T) {
		mockBus := new(mocks.MockBus)
		// Create a valid amount for the test
		amount, err := money.New(100, "USD")
		assert.NoError(t, err)

		// Create a test account ID and user ID
		accountID := uuid.New()
		userID := uuid.New()

		req := events.WithdrawRequestedEvent{
			FlowEvent: events.FlowEvent{
				AccountID:     accountID,
				UserID:        userID,
				CorrelationID: uuid.New(),
			},
			ID:     uuid.New(),
			Amount: amount,
		}

		mockUOW := new(mocks.MockUnitOfWork)
		mockAccountRepo := new(mocks.MockAccountRepository)

		// Set up the account repository mock
		mockUOW.On("GetRepository", (*account.Repository)(nil)).Return(mockAccountRepo, nil).Once()

		// Create a test account with the same user ID as in the request
		testAccount := &dto.AccountRead{
			ID:       req.AccountID,
			UserID:   req.UserID,
			Currency: "USD",
			Balance:  1000, // Sufficient balance
		}

		mockAccountRepo.On("Get", ctx, req.AccountID).Return(testAccount, nil).Once()

		// Expect the WithdrawValidatedEvent to be emitted
		mockBus.On("Emit", ctx, mock.MatchedBy(func(event interface{}) bool {
			e, ok := event.(events.WithdrawValidatedEvent)
			if !ok {
				return false
			}
			return e.AccountID == req.AccountID && e.UserID == req.UserID
		})).Return(nil).Once()

		handler := withdraw.Validation(mockBus, mockUOW, logger)
		handlerErr := handler(ctx, req)

		// The validation handler returns an error when it fails to get the repository
		assert.Error(t, handlerErr)
		assert.Equal(t, "failed to get repo", handlerErr.Error())

		// The account repository's Get method should not have been called
		mockAccountRepo.AssertNotCalled(t, "Get", ctx, req.AccountID)
		// The bus should not have been called since we failed to get the repository
		mockBus.AssertNotCalled(t, "Emit", ctx, mock.Anything)
	})

	t.Run("should emit WithdrawFailedEvent on invalid request", func(t *testing.T) {
		mockUOW := new(mocks.MockUnitOfWork)

		// No specific mocks for GetRepository or Get needed here as the handler should not call them
		// for an invalid request that's caught early.

		mockBus := new(mocks.MockBus)
		mockBus.On("Emit", ctx, mock.AnythingOfType("events.WithdrawFailedEvent")).Return(nil).Once()

		handler := withdraw.Validation(mockBus, mockUOW, logger)
		amount, _ := money.New(0, "USD")
		req := events.WithdrawRequestedEvent{
			FlowEvent: events.FlowEvent{
				AccountID: uuid.Nil,
			},
			ID:     uuid.New(),
			Amount: amount,
		}

		err := handler(ctx, req)

		assert.NoError(t, err)
		mockBus.AssertExpectations(t)
	})
}
