package payment

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/google/uuid"
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

	t.Run("handles repository error", func(t *testing.T) {
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

		// Mock the Do call to execute the callback with the mock UOW
		uow.EXPECT().Do(mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			Return(errors.New("repository error")).Once()

		// Execute
		handler := HandleProcessed(uow, logger)
		err := handler(ctx, event)

		// Assert
		require.Error(t, err)
		require.Equal(t, "repository error", err.Error())
	})
}
