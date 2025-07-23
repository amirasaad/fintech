package withdraw_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/handler/account/withdraw"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBusinessValidation(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	validAmount, _ := money.New(100, "USD")

	// Create valid UUIDs for testing
	userID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	accountID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174001")
	correlationID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174003")
	transactionID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174004")
	flow := events.FlowEvent{
		UserID:        userID,
		AccountID:     accountID,
		FlowType:      "withdraw",
		CorrelationID: correlationID,
	}
	baseEvent := events.WithdrawValidatedEvent{
		WithdrawRequestedEvent: events.WithdrawRequestedEvent{
			FlowEvent:         flow,
			ID:                uuid.New(),
			Amount:            validAmount,
			BankAccountNumber: "123456789",
			Timestamp:         time.Now(),
		},
		// Don't set the Account field here as it will be set by the handler
		// after retrieving it from the repository
		TargetCurrency: "USD",
	}
	conversionDone := events.ConversionDoneEvent{
		FlowEvent:       flow,
		ID:              uuid.New(),
		RequestID:       "test-request-id",
		TransactionID:   transactionID,
		ConvertedAmount: validAmount,
		ConversionInfo: &domain.ConversionInfo{
			ConversionRate:    1,
			OriginalAmount:    validAmount.AmountFloat(),
			OriginalCurrency:  "USD",
			ConvertedCurrency: "USD",
		},
	}

	t.Run("successfully validates and emits domain op done event", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)

		// Create a test account with sufficient balance (balance in cents)
		// Make sure the account's UserID matches the event's UserID
		accDto := &dto.AccountRead{
			ID:       accountID, // Use the accountID defined at the top of the test
			UserID:   userID,    // Use the same userID defined at the top of the test
			Balance:  50000,     // $500.00 in cents
			Currency: "USD",
		}

		event := events.WithdrawBusinessValidationEvent{
			WithdrawValidatedEvent: baseEvent,
			ConversionDoneEvent:    conversionDone,
			Amount:                 validAmount,
		}

		// Set up mocks
		uow.EXPECT().GetRepository(mock.Anything).Return(accRepo, nil).Once()
		accRepo.EXPECT().Get(mock.Anything, mock.Anything).Return(accDto, nil).Once()
		bus.EXPECT().Emit(mock.Anything, mock.AnythingOfType("events.PaymentInitiationEvent")).Return(nil).Once()

		handler := withdraw.BusinessValidation(bus, uow, logger)

		err := handler(ctx, event)

		assert.NoError(t, err)
	})

	t.Run("returns with error for insufficient funds", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		repo := mocks.NewAccountRepository(t)

		// Create test event
		event := events.WithdrawBusinessValidationEvent{
			WithdrawValidatedEvent: baseEvent,
			Amount:                 validAmount,
			ConversionDoneEvent:    conversionDone,
		}

		// Create account with insufficient balance (balance in cents)
		accDto := &dto.AccountRead{
			ID:       accountID,
			UserID:   userID,
			Balance:  50,
			Currency: "USD",
		}

		// Set up mocks
		uow.EXPECT().GetRepository(mock.Anything).Return(repo, nil).Once()
		repo.EXPECT().Get(mock.Anything, mock.Anything).Return(accDto, nil).Once()

		handler := withdraw.BusinessValidation(bus, uow, logger)
		err := handler(ctx, event)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient funds")
	})

	t.Run("returns error when account not found", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		repo := mocks.NewAccountRepository(t)

		// Create test event
		event := events.WithdrawBusinessValidationEvent{
			WithdrawValidatedEvent: baseEvent,
			Amount:                 validAmount,
			ConversionDoneEvent:    conversionDone,
		}

		// Set up mocks
		uow.EXPECT().GetRepository(mock.Anything).Return(repo, nil).Once()
		repo.EXPECT().Get(ctx, mock.Anything).Return(nil, errors.New("account not found")).Once()

		handler := withdraw.BusinessValidation(bus, uow, logger)
		err := handler(ctx, event)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "account not found")
	})

	t.Run("returns error on repository failure", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		// Create test event
		event := events.WithdrawBusinessValidationEvent{
			WithdrawValidatedEvent: baseEvent,
			Amount:                 validAmount,
			ConversionDoneEvent:    conversionDone,
		}

		// Set up mocks to return error when getting repository
		expectedErr := errors.New("repository error")
		uow.EXPECT().GetRepository(mock.Anything).Return(nil, expectedErr).Once()

		handler := withdraw.BusinessValidation(bus, uow, logger)
		err := handler(ctx, event)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}
