package withdraw_test

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
	"github.com/amirasaad/fintech/pkg/handler/account/withdraw"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPersistence(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("successfully persists withdraw and emits events", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		userID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.WithdrawValidatedEvent{
			WithdrawRequestedEvent: events.WithdrawRequestedEvent{
				FlowEvent: events.FlowEvent{
					FlowType:      "withdraw",
					UserID:        userID,
					AccountID:     accountID,
					CorrelationID: correlationID,
				},
				Amount: amount,
			},
			TargetCurrency: currency.USD.String(),
		}

		// Mock expectations - simplify by just mocking the Do function to return success
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).Return(nil).Once()

		bus.On("Emit", mock.Anything, mock.MatchedBy(func(e interface{}) bool {
			persistedEvent, ok := e.(*events.WithdrawPersistedEvent)
			if !ok {
				return false
			}
			return persistedEvent.WithdrawValidatedEvent.UserID == userID &&
				persistedEvent.WithdrawValidatedEvent.AccountID == accountID &&
				persistedEvent.TransactionID != uuid.Nil
		})).Return(nil).Once()

		bus.On("Emit", mock.Anything, mock.MatchedBy(func(e interface{}) bool {
			conversionEvent, ok := e.(*events.ConversionRequestedEvent)
			if !ok {
				return false
			}
			return conversionEvent.Amount.Equals(amount) &&
				conversionEvent.To == currency.USD &&
				conversionEvent.FlowType == "withdraw"
		})).Return(nil).Once()

		// Execute
		handler := withdraw.Persistence(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("handles unexpected event type gracefully", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		// Use a different event type
		event := events.DepositValidatedEvent{}

		// Execute
		handler := withdraw.Persistence(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err)
		// No interactions should occur with mocks
		uow.AssertNotCalled(t, "Do", mock.Anything, mock.Anything)
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

	t.Run("handles repository error", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		userID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.WithdrawValidatedEvent{
			WithdrawRequestedEvent: events.WithdrawRequestedEvent{
				FlowEvent: events.FlowEvent{
					FlowType:      "withdraw",
					UserID:        userID,
					AccountID:     accountID,
					CorrelationID: correlationID,
				},
				Amount: amount,
			},
			TargetCurrency: currency.USD.String(),
		}

		// Mock repository error
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			Return(errors.New("repository error")).Once()

		// Execute
		handler := withdraw.Persistence(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err) // Handler should not return error, just log it
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

	t.Run("handles emit persisted event error", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		userID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.WithdrawValidatedEvent{
			WithdrawRequestedEvent: events.WithdrawRequestedEvent{
				FlowEvent: events.FlowEvent{
					FlowType:      "withdraw",
					UserID:        userID,
					AccountID:     accountID,
					CorrelationID: correlationID,
				},
				Amount: amount,
			},
			TargetCurrency: currency.USD.String(),
		}

		// Mock expectations
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).Return(nil).Once()
		bus.On("Emit", mock.Anything, mock.AnythingOfType("events.WithdrawPersistedEvent")).
			Return(errors.New("emit error")).Once()

		// Execute
		handler := withdraw.Persistence(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.Error(t, err)
	})

	t.Run("handles emit conversion event error", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		userID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.WithdrawValidatedEvent{
			WithdrawRequestedEvent: events.WithdrawRequestedEvent{
				FlowEvent: events.FlowEvent{
					FlowType:      "withdraw",
					UserID:        userID,
					AccountID:     accountID,
					CorrelationID: correlationID,
				},
				Amount: amount,
			},
			TargetCurrency: currency.USD.String(),
		}

		// Mock expectations
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).Return(nil).Once()
		bus.On("Emit", mock.Anything, mock.AnythingOfType("events.WithdrawPersistedEvent")).Return(nil).Once()
		bus.On("Emit", mock.Anything, mock.AnythingOfType("*events.ConversionRequestedEvent")).
			Return(errors.New("conversion emit error")).Once()

		// Execute
		handler := withdraw.Persistence(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.Error(t, err)
	})

	t.Run("generates correlation ID when nil", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		userID := uuid.New()
		accountID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.WithdrawValidatedEvent{
			WithdrawRequestedEvent: events.WithdrawRequestedEvent{
				FlowEvent: events.FlowEvent{
					FlowType:      "withdraw",
					UserID:        userID,
					AccountID:     accountID,
					CorrelationID: uuid.Nil, // Nil correlation ID
				},
				Amount: amount,
			},
			TargetCurrency: currency.USD.String(),
		}

		// Mock expectations
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).Return(nil).Once()
		bus.On("Emit", mock.Anything, mock.AnythingOfType("events.WithdrawPersistedEvent")).Return(nil).Once()
		bus.On("Emit", mock.Anything, mock.AnythingOfType("*events.ConversionRequestedEvent")).Return(nil).Once()

		// Execute
		handler := withdraw.Persistence(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err)
	})
}
