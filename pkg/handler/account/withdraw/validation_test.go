package withdraw_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/handler/account/withdraw"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestValidation(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	validAmount, _ := money.New(100, "USD")

	baseEvent := events.WithdrawRequestedEvent{
		AccountID: uuid.New(),
		UserID:    uuid.New(),
		Amount:    validAmount,
	}

	t.Run("successfully validates and emits event", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		bus.On("Emit", ctx, mock.AnythingOfType("events.WithdrawValidatedEvent")).Return(nil)

		handler := withdraw.Validation(bus, logger)
		err := handler(ctx, baseEvent)

		assert.NoError(t, err)
		bus.AssertExpectations(t)
	})

	t.Run("emits failed event for invalid request data", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		bus.On("Emit", ctx, mock.AnythingOfType("events.WithdrawFailedEvent")).Return(nil)

		invalidEvent := baseEvent
		invalidEvent.AccountID = uuid.Nil

		handler := withdraw.Validation(bus, logger)
		err := handler(ctx, invalidEvent)

		assert.NoError(t, err)
		bus.AssertExpectations(t)
	})

	t.Run("discards malformed event", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		handler := withdraw.Validation(bus, logger)
		err := handler(ctx, "not a real event")

		assert.NoError(t, err)
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})
}
