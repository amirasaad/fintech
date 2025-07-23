package withdraw

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

func TestValidation(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("should emit WithdrawValidatedEvent on valid request", func(t *testing.T) {
		// Setup
		mockBus := mocks.NewMockBus(t)
		mockUoW := mocks.NewMockUnitOfWork(t)
		mockAccRepo := mocks.NewAccountRepository(t)

		userID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.WithdrawRequestedEvent{
			FlowEvent: events.FlowEvent{
				FlowType:      "withdraw",
				UserID:        userID,
				AccountID:     accountID,
				CorrelationID: correlationID,
			},
			ID:     uuid.New(),
			Amount: amount,
		}

		accRead := &dto.AccountRead{
			ID:       accountID,
			UserID:   userID,
			Balance:  1000.0,
			Currency: "USD",
		}

		// Mock expectations
		mockUoW.On("GetRepository", mock.Anything).Return(mockAccRepo, nil).Once()
		mockAccRepo.On("Get", ctx, accountID).Return(accRead, nil).Once()
		mockBus.On("Emit", mock.Anything, mock.MatchedBy(func(e interface{}) bool {
			validatedEvent, ok := e.(events.WithdrawValidatedEvent)
			if !ok {
				return false
			}
			return validatedEvent.WithdrawRequestedEvent.UserID == userID &&
				validatedEvent.WithdrawRequestedEvent.AccountID == accountID
		})).Return(nil).Once()

		// Execute
		handler := Validation(mockBus, mockUoW, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("should emit WithdrawFailedEvent on invalid request", func(t *testing.T) {
		// Setup
		mockBus := mocks.NewMockBus(t)
		mockUoW := mocks.NewMockUnitOfWork(t)

		// Invalid event with nil UUIDs
		event := events.WithdrawRequestedEvent{
			ID: uuid.New(),
		}

		// Mock expectations for failed event
		mockBus.On("Emit", mock.Anything, mock.MatchedBy(func(e interface{}) bool {
			_, ok := e.(events.WithdrawFailedEvent)
			return ok
		})).Return(nil).Once()

		// Execute
		handler := Validation(mockBus, mockUoW, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err) // Handler should not return error, just emit failed event
	})

	t.Run("should handle account not found", func(t *testing.T) {
		// Setup
		mockBus := mocks.NewMockBus(t)
		mockUoW := mocks.NewMockUnitOfWork(t)
		mockAccRepo := mocks.NewAccountRepository(t)

		userID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.WithdrawRequestedEvent{
			FlowEvent: events.FlowEvent{
				FlowType:      "withdraw",
				UserID:        userID,
				AccountID:     accountID,
				CorrelationID: correlationID,
			},
			ID:     uuid.New(),
			Amount: amount,
		}

		// Mock expectations
		mockUoW.On("GetRepository", mock.Anything).Return(mockAccRepo, nil).Once()
		mockAccRepo.On("Get", ctx, accountID).Return(nil, errors.New("account not found")).Once()
		mockBus.On("Emit", mock.Anything, mock.MatchedBy(func(e interface{}) bool {
			failedEvent, ok := e.(events.WithdrawFailedEvent)
			if !ok {
				return false
			}
			return failedEvent.Reason == "Account not found"
		})).Return(nil).Once()

		// Execute
		handler := Validation(mockBus, mockUoW, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err)
	})
}
