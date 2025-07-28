package transfer_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/handler/account/transfer"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestValidation(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("successfully validates transfer and emits validated event", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)

		userID := uuid.New()
		accountID := uuid.New()
		destAccountID := uuid.New()
		amount, _ := money.New(100, currency.USD)
		event := events.NewTransferRequestedEvent(
			userID,
			accountID,
			uuid.New(),
			events.WithTransferDestAccountID(destAccountID),
			events.WithTransferRequestedAmount(amount),
		)

		bus.On("Emit", mock.Anything, mock.MatchedBy(func(e interface{}) bool {
			validatedEvent, ok := e.(*events.TransferValidatedEvent)
			if !ok {
				return false
			}
			return validatedEvent.TransferRequestedEvent.UserID == userID &&
				validatedEvent.TransferRequestedEvent.AccountID == accountID &&
				validatedEvent.TransferRequestedEvent.DestAccountID == destAccountID
		})).Return(nil).Once()

		// Execute
		handler := transfer.Validation(bus, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("handles malformed event gracefully", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)

		// Event with missing required fields
		event := events.NewTransferRequestedEvent(
			uuid.Nil,
			uuid.Nil,
			uuid.New(),
			events.WithTransferDestAccountID(uuid.Nil),
			events.WithTransferRequestedAmount(money.Money{}),
		)

		// Execute
		handler := transfer.Validation(bus, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err) // Handler should not return error, just log and discard
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

	t.Run("handles unexpected event type gracefully", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)

		// Use a different event type
		event := events.DepositRequestedEvent{}

		// Execute
		handler := transfer.Validation(bus, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err)
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

	t.Run("handles negative amount", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)

		userID := uuid.New()
		accountID := uuid.New()
		destAccountID := uuid.New()
		amount, _ := money.New(-100, currency.USD)
		event := events.NewTransferRequestedEvent(
			userID,
			accountID,
			uuid.New(),
			events.WithTransferDestAccountID(destAccountID),
			events.WithTransferRequestedAmount(amount),
		)

		// Execute
		handler := transfer.Validation(bus, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err) // Handler should not return error, just log and discard
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})
}
