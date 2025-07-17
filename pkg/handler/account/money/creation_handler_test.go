package money

import (
	"context"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMoneyCreationHandler_PublishesMoneyCreatedEvent(t *testing.T) {
	tests := []struct {
		name       string
		input      events.DepositValidatedEvent
		expectEvt  bool
		setupMocks func(bus *mocks.MockEventBus)
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
			setupMocks: func(bus *mocks.MockEventBus) {
				bus.On("Publish", mock.Anything, mock.AnythingOfType("events.MoneyCreatedEvent")).Return(nil)
			},
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
			expectEvt:  false,
			setupMocks: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := mocks.NewMockEventBus(t)
			if tc.setupMocks != nil {
				tc.setupMocks(bus)
			}
			handler := MoneyCreationHandler(bus)
			handler(context.Background(), tc.input)
			if tc.expectEvt {
				assert.True(t, bus.AssertCalled(t, "Publish", mock.Anything, mock.AnythingOfType("events.MoneyCreatedEvent")), "should publish MoneyCreatedEvent")
			} else {
				bus.AssertNotCalled(t, "Publish", mock.Anything, mock.AnythingOfType("events.MoneyCreatedEvent"))
			}
		})
	}
}
