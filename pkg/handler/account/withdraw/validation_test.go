package withdraw_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/handler/account/withdraw"
	accountRepo "github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestValidation(t *testing.T) {
	ctx := context.Background()
	// Create a logger that outputs to stderr for debugging
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	logger.Info("Starting withdraw validation tests")

	t.Run("should emit WithdrawValidatedEvent on valid request", func(t *testing.T) {
		// Setup
		mockBus := mocks.NewMockBus(t)
		mockUoW := mocks.NewMockUnitOfWork(t)
		mockAccRepo := mocks.NewAccountRepository(t)

		userID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := &events.WithdrawRequestedEvent{
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
		mockUoW.On("GetRepository", (*accountRepo.Repository)(nil)).Return(mockAccRepo, nil).Once()
		mockAccRepo.On("Get", ctx, accountID).Return(accRead, nil).Once()
		mockBus.On("Emit", ctx, mock.MatchedBy(func(e any) bool {
			validatedEvent, ok := e.(*events.WithdrawValidatedEvent)
			if !ok {
				return false
			}
			// Check that the FlowEvent fields are set correctly
			return validatedEvent.UserID == userID &&
				validatedEvent.AccountID == accountID &&
				validatedEvent.CorrelationID == correlationID &&
				validatedEvent.FlowType == "withdraw"
		})).Return(nil).Once()

		// Execute
		handler := withdraw.Validation(mockBus, mockUoW, logger)
		logger.Debug("Calling handler with event", "event", event)
		err := handler(ctx, event)
		logger.Debug("Handler returned", "error", err)

		// Assert
		assert.NoError(t, err)

		// Verify all expectations were met
		mockBus.AssertExpectations(t)
		mockUoW.AssertExpectations(t)
		mockAccRepo.AssertExpectations(t)
	})

	t.Run("should emit WithdrawFailedEvent on invalid request", func(t *testing.T) {
		// Setup
		mockBus := mocks.NewMockBus(t)
		mockUoW := mocks.NewMockUnitOfWork(t)

		// Invalid event with nil UUIDs
		event := &events.WithdrawRequestedEvent{
			FlowEvent: events.FlowEvent{
				FlowType: "withdraw",
			},
			ID: uuid.New(),
		}

		// Mock expectations for failed event
		mockBus.On("Emit", ctx, mock.MatchedBy(func(e interface{}) bool {
			failedEvent, ok := e.(*events.WithdrawFailedEvent)
			if !ok {
				return false
			}
			return failedEvent.FlowType == "withdraw"
		})).Return(nil).Once()

		// Execute
		handler := withdraw.Validation(mockBus, mockUoW, logger)
		logger.Debug("Calling handler with event", "event", event)
		err := handler(ctx, event)
		logger.Debug("Handler returned", "error", err)

		// Assert
		assert.NoError(t, err)

		// Verify all expectations were met
		mockBus.AssertExpectations(t)
		mockUoW.AssertExpectations(t)
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

		event := &events.WithdrawRequestedEvent{
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
		mockUoW.On("GetRepository", (*accountRepo.Repository)(nil)).Return(mockAccRepo, nil).Once()
		mockAccRepo.On("Get", ctx, accountID).Return(nil, errors.New("account not found")).Once()
		mockBus.On("Emit", ctx, mock.MatchedBy(func(e interface{}) bool {
			failedEvent, ok := e.(*events.WithdrawFailedEvent)
			if !ok {
				return false
			}
			return failedEvent.FlowType == "withdraw" && failedEvent.Reason == "Account not found"
		})).Return(nil).Once()

		// Execute
		handler := withdraw.Validation(mockBus, mockUoW, logger)
		logger.Debug("Calling handler with event", "event", event)
		err := handler(ctx, event)
		logger.Debug("Handler returned", "error", err)

		// Assert
		assert.NoError(t, err)

		// Verify all expectations were met
		mockBus.AssertExpectations(t)
		mockUoW.AssertExpectations(t)
		mockAccRepo.AssertExpectations(t)
	})
}
