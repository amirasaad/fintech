package account

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// mockEventBus is now defined in testutils_test.go for reuse.

func TestDepositValidationHandler_PublishesValidatedEvent(t *testing.T) {
	tests := []struct {
		name      string
		input     events.DepositRequestedEvent
		expectEvt bool
	}{
		{
			name: "valid input",
			input: events.DepositRequestedEvent{
				UserID:    uuid.NewString(),
				AccountID: uuid.NewString(),
				Amount:    100.0,
				Currency:  "USD",
			},
			expectEvt: true,
		},
		{
			name: "missing user",
			input: events.DepositRequestedEvent{
				UserID:    "",
				AccountID: uuid.NewString(),
				Amount:    100.0,
				Currency:  "USD",
			},
			expectEvt: false,
		},
		{
			name: "zero amount",
			input: events.DepositRequestedEvent{
				UserID:    uuid.NewString(),
				AccountID: uuid.NewString(),
				Amount:    0,
				Currency:  "USD",
			},
			expectEvt: false,
		},
		{
			name: "missing currency: defaulted to DefaultCurrency",
			input: events.DepositRequestedEvent{
				UserID:    uuid.NewString(),
				AccountID: uuid.NewString(),
				Amount:    100.0,
				Currency:  "",
			},
			expectEvt: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := &mockEventBus{}
			handler := DepositValidationHandler(bus, slog.New(slog.NewTextHandler(io.Discard, nil)))
			handler(context.Background(), tc.input)
			if tc.expectEvt {
				// assert.NotEmpty(t, bus.published, "should publish an event")
				_, ok := bus.published[0].(events.DepositValidatedEvent)
				assert.True(t, ok, "should publish DepositValidatedEvent")
			} else {
				assert.Empty(t, bus.published, "should not publish event")
			}
		})
	}
}
