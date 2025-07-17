package account

import (
	"context"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/stretchr/testify/assert"
)

// mockEventBus is defined in testutils_test.go for reuse.

func TestMoneyCreationHandler_PublishesMoneyCreatedEvent(t *testing.T) {
	tests := []struct {
		name      string
		input     events.DepositValidatedEvent
		expectEvt bool
	}{
		{
			name: "valid input",
			input: events.DepositValidatedEvent{
				DepositRequestedEvent: events.DepositRequestedEvent{
					UserID:    "user-1",
					AccountID: "acc-1",
					Amount:    100.0,
					Currency:  "USD",
				},
			},
			expectEvt: true,
		},
		{
			name: "zero amount",
			input: events.DepositValidatedEvent{
				DepositRequestedEvent: events.DepositRequestedEvent{
					UserID:    "user-1",
					AccountID: "acc-1",
					Amount:    0,
					Currency:  "USD",
				},
			},
			expectEvt: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := &mockEventBus{}
			handler := MoneyCreationHandler(bus)
			handler(context.Background(), tc.input)
			if tc.expectEvt {
				assert.NotEmpty(t, bus.published, "should publish an event")
				_, ok := bus.published[0].(events.MoneyCreatedEvent)
				assert.True(t, ok, "should publish MoneyCreatedEvent")
			} else {
				assert.Empty(t, bus.published, "should not publish event")
			}
		})
	}
}
