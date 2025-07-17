package deposit

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockEventBus is now defined in testutils_test.go for reuse.

func TestDepositValidationHandler_PublishesValidatedEvent(t *testing.T) {
	tests := []struct {
		name       string
		input      events.DepositRequestedEvent
		expectEvt  bool
		setupMocks func(bus *mocks.MockEventBus)
	}{
		{
			name:      "valid input",
			input:     events.DepositRequestedEvent{UserID: uuid.NewString(), AccountID: uuid.NewString(), Amount: 100.0, Currency: "USD"},
			expectEvt: true,
			setupMocks: func(bus *mocks.MockEventBus) {
				bus.On("Publish", mock.Anything, mock.AnythingOfType("events.DepositValidatedEvent")).Return(nil)
			},
		},
		{
			name:       "missing user",
			input:      events.DepositRequestedEvent{UserID: "", AccountID: uuid.NewString(), Amount: 100.0, Currency: "USD"},
			expectEvt:  false,
			setupMocks: nil,
		},
		{
			name:       "zero amount",
			input:      events.DepositRequestedEvent{UserID: uuid.NewString(), AccountID: uuid.NewString(), Amount: 0, Currency: "USD"},
			expectEvt:  false,
			setupMocks: nil,
		},
		{
			name:      "missing currency: defaulted to DefaultCurrency",
			input:     events.DepositRequestedEvent{UserID: uuid.NewString(), AccountID: uuid.NewString(), Amount: 100.0, Currency: ""},
			expectEvt: true,
			setupMocks: func(bus *mocks.MockEventBus) {
				bus.On("Publish", mock.Anything, mock.AnythingOfType("events.DepositValidatedEvent")).Return(nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := mocks.NewMockEventBus(t)
			if tc.setupMocks != nil {
				tc.setupMocks(bus)
			}
			handler := DepositValidationHandler(bus, slog.New(slog.NewTextHandler(io.Discard, nil)))
			handler(context.Background(), tc.input)
			if tc.expectEvt {
				assert.True(t, bus.AssertCalled(t, "Publish", mock.Anything, mock.AnythingOfType("events.DepositValidatedEvent")), "should publish DepositValidatedEvent")
			} else {
				bus.AssertNotCalled(t, "Publish", mock.Anything, mock.AnythingOfType("events.DepositValidatedEvent"))
			}
		})
	}
}
