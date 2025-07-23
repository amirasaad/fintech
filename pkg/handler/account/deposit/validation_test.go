package deposit

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

	t.Run("successfully validates deposit and emits validated event", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)

		userID := uuid.New()
		accountID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.DepositRequestedEvent{
			FlowEvent: events.FlowEvent{
				FlowType:      "deposit",
				UserID:        userID,
				AccountID:     accountID,
				CorrelationID: uuid.New(),
			},
			Amount: amount,
		}

		accRead := &dto.AccountRead{
			ID:       accountID,
			UserID:   userID,
			Balance:  1000.0,
			Currency: "USD",
		}

		// Mock expectations
		uow.On("GetRepository", mock.Anything).Return(accRepo, nil).Once()
		accRepo.On("Get", mock.Anything, accountID).Return(accRead, nil).Once()
		bus.On("Emit", mock.Anything, mock.MatchedBy(func(e interface{}) bool {
			validatedEvent, ok := e.(events.DepositValidatedEvent)
			if !ok {
				return false
			}
			return validatedEvent.DepositRequestedEvent.UserID == userID &&
				validatedEvent.DepositRequestedEvent.AccountID == accountID &&
				validatedEvent.Account != nil
		})).Return(nil).Once()

		// Execute
		handler := Validation(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("handles unexpected event type gracefully", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		// Use a different event type
		event := events.WithdrawRequestedEvent{}

		// Execute
		handler := Validation(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err)
		// No interactions should occur with mocks
		uow.AssertNotCalled(t, "GetRepository", mock.Anything)
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

	t.Run("handles repository error", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		userID := uuid.New()
		accountID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.DepositRequestedEvent{
			FlowEvent: events.FlowEvent{
				FlowType:      "deposit",
				UserID:        userID,
				AccountID:     accountID,
				CorrelationID: uuid.New(),
			},
			Amount: amount,
		}

		// Mock repository error
		uow.On("GetRepository", mock.Anything).Return(nil, errors.New("repository error")).Once()

		// Execute
		handler := Validation(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err) // Handler should not return error, just log it
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

	t.Run("handles account not found", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)

		userID := uuid.New()
		accountID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.DepositRequestedEvent{
			FlowEvent: events.FlowEvent{
				FlowType:      "deposit",
				UserID:        userID,
				AccountID:     accountID,
				CorrelationID: uuid.New(),
			},
			Amount: amount,
		}

		// Mock expectations
		uow.On("GetRepository", mock.Anything).Return(accRepo, nil).Once()
		accRepo.On("Get", mock.Anything, accountID).Return(nil, errors.New("account not found")).Once()

		// Execute
		handler := Validation(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err) // Handler should not return error, just log it
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

	t.Run("handles account validation failure", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)

		userID := uuid.New()
		accountID := uuid.New()
		// Use a negative amount to trigger validation failure
		amount, _ := money.New(-100, currency.USD)

		event := events.DepositRequestedEvent{
			FlowEvent: events.FlowEvent{
				FlowType:      "deposit",
				UserID:        userID,
				AccountID:     accountID,
				CorrelationID: uuid.New(),
			},
			Amount: amount,
		}

		accRead := &dto.AccountRead{
			ID:       accountID,
			UserID:   userID,
			Balance:  1000.0,
			Currency: "USD",
		}

		// Mock expectations
		uow.On("GetRepository", mock.Anything).Return(accRepo, nil).Once()
		accRepo.On("Get", mock.Anything, accountID).Return(accRead, nil).Once()

		// Execute
		handler := Validation(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.Error(t, err) // Should return validation error
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

	t.Run("handles wrong user ID", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)

		userID := uuid.New()
		wrongUserID := uuid.New()
		accountID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.DepositRequestedEvent{
			FlowEvent: events.FlowEvent{
				FlowType:      "deposit",
				UserID:        wrongUserID, // Different user ID
				AccountID:     accountID,
				CorrelationID: uuid.New(),
			},
			Amount: amount,
		}

		accRead := &dto.AccountRead{
			ID:       accountID,
			UserID:   userID, // Account belongs to different user
			Balance:  1000.0,
			Currency: "USD",
		}

		// Mock expectations
		uow.On("GetRepository", mock.Anything).Return(accRepo, nil).Once()
		accRepo.On("Get", mock.Anything, accountID).Return(accRead, nil).Once()

		// Execute
		handler := Validation(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.Error(t, err) // Should return validation error for wrong user
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})
}
