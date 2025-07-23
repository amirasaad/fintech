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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPersistence(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("successfully persists payment ID", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		transactionID := uuid.New()
		paymentID := "pm_12345"

		event := events.PaymentInitiatedEvent{
			TransactionID: transactionID,
			PaymentID:     paymentID,
		}

		// Mock expectations
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).Return(nil).Once()
		// bus.On("Emit", mock.Anything, mock.Anything).Return(nil).Once()

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
		event := events.DepositRequestedEvent{}

		// Execute
		handler := Persistence(bus, uow, logger)
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

		transactionID := uuid.New()
		paymentID := "pm_12345"

		event := events.PaymentInitiatedEvent{
			TransactionID: transactionID,
			PaymentID:     paymentID,
		}

		// Mock repository error
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			Return(errors.New("repository error")).Once()

		// Execute
		handler := Persistence(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.Error(t, err)
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})
}
