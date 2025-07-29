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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBusinessValidation(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("successfully validates and emits payment initiation event", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)

		userID := uuid.New()
		transactionID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		// Create a new account ID for the test
		accountID := uuid.New()
		event := NewValidDepositBusinessValidationEvent(userID, accountID, transactionID, amount)
		// Use the account ID from the event's FlowEvent
		accountID = event.FlowEvent.AccountID

		accRead := &dto.AccountRead{
			ID:       accountID,
			UserID:   userID,
			Balance:  1000.0,
			Currency: "USD",
		}

		// Mock expectations
		uow.On("GetRepository", mock.Anything).Return(accRepo, nil).Once()
		accRepo.On("Get", context.Background(), accountID).Return(accRead, nil).Once()
		bus.On("Emit", context.Background(), mock.MatchedBy(func(e interface{}) bool {
			paymentEvent, ok := e.(*events.PaymentInitiationEvent)
			if !ok {
				t.Logf("Expected *events.PaymentInitiationEvent, got %T", e)
				return false
			}

			// Log the actual values for debugging
			t.Logf("PaymentEvent: TransactionID=%v, Amount=%v, Account=%+v",
				paymentEvent.TransactionID,
				paymentEvent.Amount,
				paymentEvent.Account)

			// Check that the required fields match
			if paymentEvent.TransactionID != transactionID {
				t.Logf("TransactionID mismatch: expected %v, got %v", transactionID, paymentEvent.TransactionID)
				return false
			}

			if !paymentEvent.Amount.Equals(amount) {
				t.Logf("Amount mismatch: expected %v, got %v", amount, paymentEvent.Amount)
				return false
			}

			if paymentEvent.Account == nil {
				t.Log("Account is nil")
				return false
			}

			if paymentEvent.Account.ID != accountID {
				t.Logf("Account ID mismatch: expected %v, got %v", accountID, paymentEvent.Account.ID)
				return false
			}

			if paymentEvent.Account.UserID != userID {
				t.Logf("UserID mismatch: expected %v, got %v", userID, paymentEvent.Account.UserID)
				return false
			}

			return true
		})).Return(nil).Once()

		// Execute
		handler := deposit.BusinessValidation(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("handles unexpected event type gracefully", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		// No mock expectations for unexpected event type

		// Use a different event type
		event := events.WithdrawBusinessValidationEvent{}

		// Execute
		handler := deposit.BusinessValidation(bus, uow, logger)
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
		transactionID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.NewDepositBusinessValidationEvent(
			userID, accountID, transactionID,
			events.WithBusinessValidationAmount(amount),
		)

		// Mock repository error
		uow.On("GetRepository", mock.Anything).Return(nil, errors.New("repository error")).Once()

		// Execute
		handler := deposit.BusinessValidation(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.Error(t, err)
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

	t.Run("handles invalid repository type", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		userID := uuid.New()
		accountID := uuid.New()
		transactionID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.NewDepositBusinessValidationEvent(
			userID, accountID, transactionID,
			events.WithBusinessValidationAmount(amount),
		)

		// Mock returning wrong repository type
		uow.On("GetRepository", mock.Anything).Return("wrong type", nil).Once()

		// Execute
		handler := deposit.BusinessValidation(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.Error(t, err)
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

}
