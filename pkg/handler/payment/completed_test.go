package payment

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockEvent struct{}

func (m *mockEvent) Type() string { return "mockEvent" }

func TestCompletedHandler(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	bus := mocks.NewMockBus(t)
	mUow := mocks.NewMockUnitOfWork(t)

	validEvent := events.NewPaymentCompleted(
		events.FlowEvent{
			ID:            uuid.New(),
			FlowType:      "payment",
			UserID:        uuid.Nil,
			AccountID:     uuid.Nil,
			CorrelationID: uuid.Nil,
		},
		events.WithPaymentID("pay_123"),
		events.WithCorrelationID(uuid.New()),
	)

	t.Run("returns nil for incorrect event type", func(t *testing.T) {
		h := HandleCompleted(bus, mUow, logger)
		err := h(ctx, &mockEvent{})
		assert.NoError(t, err)
	})

	t.Run("handles error from unit of work", func(t *testing.T) {
		h := HandleCompleted(bus, mUow, logger)
		mUow.On("Do", ctx, mock.Anything).Return(errors.New("uow error")).Once()
		err := h(ctx, validEvent)
		assert.Error(t, err)
	})

	t.Run("handles successful event", func(t *testing.T) {
		h := HandleCompleted(bus, mUow, logger)
		mUow.On("Do", ctx, mock.Anything).Return(nil).Once()
		err := h(ctx, validEvent)
		assert.NoError(t, err)
	})
}
