package transfer

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

// Helper to create a valid event for tests
func newValidConversionDoneEvent(t *testing.T) events.TransferBusinessValidatedEvent {
	t.Helper()
	userID := uuid.New()
	accountID := uuid.New()
	correlationID := uuid.New()
	amount, err := money.New(100, currency.USD)
	assert.NoError(t, err)

	transferEvent := events.TransferRequestedEvent{
		FlowEvent: events.FlowEvent{
			FlowType:      "transfer",
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: correlationID,
		},
		ID:             uuid.New(),
		Amount:         amount,
		DestAccountID:  uuid.New(),
		ReceiverUserID: uuid.New(),
	}

	return events.TransferBusinessValidatedEvent{
		TransferValidatedEvent: events.TransferValidatedEvent{
			TransferRequestedEvent: transferEvent,
		},
		ConversionDoneEvent: events.ConversionDoneEvent{
			FlowEvent: events.FlowEvent{
				FlowType:      "transfer",
				UserID:        userID,
				AccountID:     accountID,
				CorrelationID: correlationID,
			},
			RequestID:       "test-request-id",
			TransactionID:   uuid.New(),
			ConvertedAmount: amount,
		},
	}
}

func TestBusinessValidation(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("successfully validates and emits domain op done event", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)
		event := newValidConversionDoneEvent(t)

		// Create a sufficient balance for the test
		sufficientBalance, err := money.New(20000, currency.USD)
		if !assert.NoError(t, err) {
			return
		}

		uow.On("GetRepository", mock.Anything).Return(accRepo, nil).Once()
		accRepo.On("Get", mock.Anything, event.AccountID).Return(&dto.AccountRead{
			ID:       event.AccountID,
			UserID:   event.UserID,
			Balance:  sufficientBalance.AmountFloat(),
			Currency: sufficientBalance.Currency().String(),
		}, nil).Once()
		bus.On("Emit", mock.Anything, mock.MatchedBy(func(e interface{}) bool {
			_, ok := e.(events.TransferDomainOpDoneEvent)
			return ok
		})).Return(nil).Once()

		handler := BusinessValidation(bus, uow, logger)
		handleErr := handler(ctx, event)

		assert.NoError(t, handleErr)
	})

	t.Run("emits failed event for insufficient funds", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)
		event := newValidConversionDoneEvent(t)

		// Create a balance that's twice the converted amount
		// Create a zero balance for insufficient funds test
		zeroAmount, err := money.New(0, event.ConvertedAmount.Currency())
		if !assert.NoError(t, err) {
			return
		}

		// Set up the mock to return the account repository
		uow.On("GetRepository", mock.Anything).Return(accRepo, nil).Once()

		// Set up the mock repository to return an account with insufficient funds
		accRepo.On("Get", mock.Anything, mock.Anything).Return(&dto.AccountRead{
			ID:       event.AccountID,
			UserID:   event.UserID,
			Balance:  zeroAmount.AmountFloat(),
			Currency: zeroAmount.Currency().String(),
		}, nil).Once()

		bus.On("Emit", mock.Anything, mock.MatchedBy(func(e interface{}) bool {
			_, ok := e.(events.TransferFailedEvent)
			return ok
		})).Return(nil).Once()

		handler := BusinessValidation(bus, uow, logger)
		err = handler(ctx, event)
		assert.NoError(t, err)
	})

	t.Run("emits failed event when account not found", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)
		event := newValidConversionDoneEvent(t)

		uow.On("GetRepository", mock.Anything).Return(accRepo, nil).Once()

		accRepo.On("Get", mock.Anything, mock.Anything).Return(nil, errors.New("not found")).Once()

		bus.On("Emit", mock.Anything, mock.MatchedBy(func(e interface{}) bool {
			_, ok := e.(events.TransferFailedEvent)
			return ok
		})).Return(nil).Once()

		handler := BusinessValidation(bus, uow, logger)
		err := handler(ctx, event)

		assert.NoError(t, err)
	})

	t.Run("returns error on repository failure", func(t *testing.T) {
		uow := mocks.NewMockUnitOfWork(t)
		event := newValidConversionDoneEvent(t)
		dbError := errors.New("database error")

		// Set up the mock to return an error when getting the repository
		uow.On("GetRepository", mock.Anything).Return(nil, dbError).Once()

		handler := BusinessValidation(mocks.NewMockBus(t), uow, logger)
		handleErr := handler(ctx, event)

		assert.ErrorIs(t, handleErr, dbError)
	})
}
