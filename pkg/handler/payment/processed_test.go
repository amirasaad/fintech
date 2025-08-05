package payment

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// testEvent is a simple implementation of the events.Event interface for testing
type testEvent struct{}

func (e *testEvent) Type() string {
	return "unexpected.event.type"
}

func TestHandleProcessed(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(
		slog.NewTextHandler(
			io.Discard,
			&slog.HandlerOptions{Level: slog.LevelError},
		),
	)

	t.Run("successfully persists payment ID", func(t *testing.T) {
		// Setup
		uow := mocks.NewUnitOfWork(t)

		transactionID := uuid.New()
		paymentID := "pm_12345"

		// Create a PaymentInitiated event first
		pi := &events.PaymentInitiated{
			FlowEvent: events.FlowEvent{
				ID:       uuid.New(),
				FlowType: "payment",
			},
			TransactionID: transactionID,
			PaymentID:     paymentID,
			Status:        "pending",
		}

		// Create the PaymentProcessed event that the handler expects
		event := events.NewPaymentProcessed(*pi)

		txRepo := mocks.NewTransactionRepository(t)

		// Mock expectations
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			Return(nil).
			Run(func(args mock.Arguments) {
				cb := args.Get(1).(func(repository.UnitOfWork) error)

				// Setup mocks that will be used inside the callback
				uow.Mock.On("GetRepository",
					(*transaction.Repository)(nil),
				).Return(
					txRepo,
					nil,
				).Once()

				// Mock the transaction repository Get call
				txRepo.On("Get", mock.Anything, transactionID).
					Return(&dto.TransactionRead{ID: transactionID}, nil).Once()

				// Mock the transaction repository Update call
				txRepo.On(
					"Update",
					mock.Anything,
					transactionID,
					mock.MatchedBy(
						func(update dto.TransactionUpdate) bool {
							return *update.PaymentID == paymentID && *update.Status == "pending"
						},
					),
				).Return(nil).Once()

				// Execute the callback
				err := cb(uow)
				assert.NoError(t, err, "callback should not return error")
			}).Once()

		// Execute
		handler := HandleProcessed(uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("handles unexpected event type gracefully", func(t *testing.T) {
		// Setup
		uow := mocks.NewUnitOfWork(t)

		// Use a different event type
		// Create a simple implementation of the events.Event interface
		event := &testEvent{}

		// Execute
		handler := HandleProcessed(uow, logger)
		err := handler(ctx, event)

		// Assert
		require.Error(t, err)
		// No interactions should occur with mocks
		uow.AssertNotCalled(t, "Do", mock.Anything, mock.Anything)
	})

	t.Run("handles_repository_error", func(t *testing.T) {
		// Setup
		uow := mocks.NewUnitOfWork(t)

		transactionID := uuid.New()
		paymentID := "pm_12345"

		event := &events.PaymentProcessed{
			PaymentInitiated: events.PaymentInitiated{
				FlowEvent: events.FlowEvent{
					ID:            uuid.New(),
					FlowType:      "payment",
					UserID:        uuid.Nil,
					AccountID:     uuid.Nil,
					CorrelationID: uuid.Nil,
				},
				TransactionID: transactionID,
				PaymentID:     paymentID,
				Status:        "initiated",
			},
		}

		// Create a mock transaction repository
		txRepo := mocks.NewTransactionRepository(t)

		// Mock the GetRepository call that will happen inside Do
		uow.On("GetRepository", (*transaction.Repository)(nil)).
			Return(txRepo, nil).Once()

		// Mock the Get call on the transaction repository
		txRepo.On("Get", mock.Anything, transactionID).
			Return(&dto.TransactionRead{
				ID:        transactionID,
				PaymentID: "",
			}, nil).Once()

		// Mock the Update call on the transaction repository
		txRepo.On(
			"Update",
			mock.Anything,
			transactionID,
			mock.MatchedBy(
				func(update dto.TransactionUpdate) bool {
					return *update.PaymentID == paymentID && *update.Status == "pending"
				},
			),
		).Return(nil).Once()

		// Mock the Do call to execute the callback with the mock UOW
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			Run(func(args mock.Arguments) {
				// The callback should be called with the UOW
				cb := args.Get(1).(func(repository.UnitOfWork) error)
				// Execute the callback with the mock UOW
				err := cb(uow)
				assert.NoError(t, err, "callback should not return error")
			}).
			Return(errors.New("repository error")).Once()

		// Execute
		handler := HandleProcessed(uow, logger)
		err := handler(ctx, event)

		// Assert
		require.Error(t, err)
		require.Equal(t, "repository error", err.Error())
	})
}
