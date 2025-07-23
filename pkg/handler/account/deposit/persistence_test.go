package deposit

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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPersistence(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("successfully persists deposit and emits persisted event", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		userID := uuid.New()
		accountID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		acc, _ := account.New().
			WithID(accountID).
			WithUserID(userID).
			WithBalance(amount.Amount()).
			WithCurrency(currency.USD).
			Build()

		event := events.DepositValidatedEvent{
			DepositRequestedEvent: events.DepositRequestedEvent{
				FlowEvent: events.FlowEvent{
					FlowType:      "deposit",
					UserID:        userID,
					AccountID:     accountID,
					CorrelationID: uuid.New(),
				},
				Amount: amount,
			},
			Account: acc,
		}

		// Mock expectations
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).Return(nil).Once()
		bus.On("Emit", mock.Anything, mock.MatchedBy(func(e interface{}) bool {
			persistedEvent, ok := e.(events.DepositPersistedEvent)
			if !ok {
				return false
			}
			return persistedEvent.DepositValidatedEvent.UserID == userID &&
				persistedEvent.DepositValidatedEvent.AccountID == accountID &&
				persistedEvent.TransactionID != uuid.Nil &&
				persistedEvent.Amount.Equals(amount)
		})).Return(nil).Once()

		// Execute
		handler := Persistence(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("handles unexpected event type gracefully", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		// Use a different event type
		event := events.WithdrawValidatedEvent{}

		// Execute
		handler := Persistence(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err)
		// No interactions should occur with mocks
		uow.AssertNotCalled(t, "Do", mock.Anything, mock.Anything)
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

	t.Run("handles persistence error", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		userID := uuid.New()
		accountID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		acc, _ := account.New().
			WithID(accountID).
			WithUserID(userID).
			WithBalance(amount.Amount()).
			WithCurrency(currency.USD).
			Build()

		event := events.DepositValidatedEvent{
			DepositRequestedEvent: events.DepositRequestedEvent{
				FlowEvent: events.FlowEvent{
					FlowType:      "deposit",
					UserID:        userID,
					AccountID:     accountID,
					CorrelationID: uuid.New(),
				},
				Amount: amount,
			},
			Account: acc,
		}

		// Mock persistence error
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).Return(errors.New("persistence error")).Once()

		// Execute
		handler := Persistence(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.Error(t, err)
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

	t.Run("handles emit error", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		userID := uuid.New()
		accountID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		acc, _ := account.New().
			WithID(accountID).
			WithUserID(userID).
			WithBalance(amount.Amount()).
			WithCurrency(currency.USD).
			Build()

		event := events.DepositValidatedEvent{
			DepositRequestedEvent: events.DepositRequestedEvent{
				FlowEvent: events.FlowEvent{
					FlowType:      "deposit",
					UserID:        userID,
					AccountID:     accountID,
					CorrelationID: uuid.New(),
				},
				Amount: amount,
			},
			Account: acc,
		}

		// Mock expectations
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).Return(nil).Once()
		bus.On("Emit", mock.Anything, mock.AnythingOfType("events.DepositPersistedEvent")).Return(errors.New("emit error")).Once()

		// Execute
		handler := Persistence(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.Error(t, err)
	})

	t.Run("does not emit conversion event when currencies match", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		userID := uuid.New()
		accountID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		// Account with same currency as deposit
		acc, _ := account.New().
			WithID(accountID).
			WithUserID(userID).
			WithBalance(amount.Amount()).
			WithCurrency(currency.USD). // Same as deposit currency
			Build()

		event := events.DepositValidatedEvent{
			DepositRequestedEvent: events.DepositRequestedEvent{
				FlowEvent: events.FlowEvent{
					FlowType:      "deposit",
					UserID:        userID,
					AccountID:     accountID,
					CorrelationID: uuid.New(),
				},
				Amount: amount,
			},
			Account: acc,
		}

		// Mock expectations - only DepositPersistedEvent should be emitted, not ConversionRequestedEvent
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).Return(nil).Once()
		bus.On("Emit", mock.Anything, mock.AnythingOfType("events.DepositPersistedEvent")).Return(nil).Once()

		// Execute
		handler := Persistence(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err)
		// Verify that ConversionRequestedEvent was not emitted
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.AnythingOfType("*events.ConversionRequestedEvent"))
	})
}
