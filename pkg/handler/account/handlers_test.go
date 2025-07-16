package account

import (
	"context"
	"testing"

	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/stretchr/testify/assert"
)

// mockEventBus is now defined in testutils_test.go for reuse.

func TestDepositValidationHandler_PublishesValidatedEvent(t *testing.T) {
	tests := []struct {
		name      string
		input     accountdomain.DepositRequestedEvent
		expectEvt bool
	}{
		{
			name: "valid input",
			input: accountdomain.DepositRequestedEvent{
				UserID:    "user-1",
				AccountID: "acc-1",
				Amount:    100.0,
				Currency:  "USD",
			},
			expectEvt: true,
		},
		{
			name: "missing user",
			input: accountdomain.DepositRequestedEvent{
				UserID:    "",
				AccountID: "acc-1",
				Amount:    100.0,
				Currency:  "USD",
			},
			expectEvt: false,
		},
		{
			name: "zero amount",
			input: accountdomain.DepositRequestedEvent{
				UserID:    "user-1",
				AccountID: "acc-1",
				Amount:    0,
				Currency:  "USD",
			},
			expectEvt: false,
		},
		{
			name: "missing currency",
			input: accountdomain.DepositRequestedEvent{
				UserID:    "user-1",
				AccountID: "acc-1",
				Amount:    100.0,
				Currency:  "",
			},
			expectEvt: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := &mockEventBus{}
			handler := DepositValidationHandler(bus)
			handler(context.Background(), tc.input)
			if tc.expectEvt {
				assert.NotEmpty(t, bus.published, "should publish an event")
				_, ok := bus.published[0].(accountdomain.DepositValidatedEvent)
				assert.True(t, ok, "should publish DepositValidatedEvent")
			} else {
				assert.Empty(t, bus.published, "should not publish event")
			}
		})
	}
}
