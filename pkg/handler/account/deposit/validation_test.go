package deposit_test

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
	"github.com/amirasaad/fintech/pkg/handler/account/deposit"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
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
		correlationID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.NewDepositRequestedEvent(userID, accountID, correlationID, events.WithDepositAmount(amount))

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
			validatedEvent, ok := e.(*events.DepositValidatedEvent)
			if !ok {
				return false
			}
			return validatedEvent.DepositRequestedEvent.UserID == userID &&
				validatedEvent.DepositRequestedEvent.AccountID == accountID
		})).Return(nil).Once()

		// Execute
		handler := deposit.Validation(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err)
		bus.AssertExpectations(t)
	})

	t.Run("handles unexpected event type gracefully", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		// Use a different event type
		event := events.WithdrawRequestedEvent{}

		// Execute
		handler := deposit.Validation(bus, uow, logger)
		err := handler(ctx, &event)

		// Assert
		assert.NoError(t, err)
		uow.AssertNotCalled(t, "GetRepository", mock.Anything)
		bus.AssertNotCalled(t, "Emit", mock.Anything)
	})

	t.Run("handles repository error on GetRepository", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		userID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.NewDepositRequestedEvent(userID, accountID, correlationID, events.WithDepositAmount(amount))

		// Mock repository error
		uow.On("GetRepository", mock.Anything).Return(nil, errors.New("repository error")).Once()

		// Execute
		handler := deposit.Validation(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err) // Handler should not return error, just log it
		bus.AssertNotCalled(t, "Emit", mock.Anything)
	})

	t.Run("handles account not found", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)

		userID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.NewDepositRequestedEvent(userID, accountID, correlationID, events.WithDepositAmount(amount))

		// Mock expectations
		uow.On("GetRepository", mock.Anything).Return(accRepo, nil).Once()
		accRepo.On("Get", mock.Anything, accountID).Return(nil, errors.New("account not found")).Once()

		// Execute
		handler := deposit.Validation(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err) // Handler should not return error, just log it
		bus.AssertNotCalled(t, "Emit", mock.Anything)
	})

	t.Run("handles account validation failure for negative amount", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)

		userID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		// Use a negative amount to trigger validation failure
		amount, _ := money.New(-100, currency.USD)

		event := events.NewDepositRequestedEvent(userID, accountID, correlationID, events.WithDepositAmount(amount))

		// Execute
		handler := deposit.Validation(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err) // Handler should not return error, just log it
		// Verify no repository calls were made
		uow.AssertNotCalled(t, "GetRepository", (*repository.AccountRepository)(nil))
		accRepo.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
		// Verify no events were emitted
		bus.AssertNotCalled(t, "Emit", mock.Anything)
	})

	t.Run("handles wrong user ID", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)

		userID := uuid.New()
		wrongUserID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.NewDepositRequestedEvent(wrongUserID, accountID, correlationID, events.WithDepositAmount(amount))

		accRead := &dto.AccountRead{
			ID:       accountID,
			UserID:   userID, // Different from event.UserID
			Balance:  1000.0,
			Currency: "USD",
		}

		// Mock expectations - these should not be called due to the validation failure
		// but we need to set them up in case the validation check happens after the repository call
		uow.On("GetRepository", (*account.Repository)(nil)).Return(accRepo, nil).Once()
		accRepo.On("Get", mock.Anything, accountID).Return(accRead, nil).Once()

		// Execute
		handler := deposit.Validation(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err) // Handler should not return error, just log it
		// Verify no events were emitted
		bus.AssertNotCalled(t, "Emit", mock.Anything)
	})
}
