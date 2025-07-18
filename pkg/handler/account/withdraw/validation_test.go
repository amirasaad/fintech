package withdraw

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

func TestWithdrawValidationHandler(t *testing.T) {
	validUserID := uuid.New()
	validAccountID := uuid.New()
	validEvent := events.WithdrawRequestedEvent{
		UserID:    validUserID,
		AccountID: validAccountID,
		Amount:    100.0,
		Currency:  "USD",
	}
	invalidEvent := events.WithdrawRequestedEvent{
		UserID:    uuid.Nil,
		AccountID: uuid.Nil,
		Amount:    -50.0,
		Currency:  "",
	}

	tests := []struct {
		name       string
		input      events.WithdrawRequestedEvent
		expectPub  bool
		setupMocks func(bus *mocks.MockEventBus)
	}{
		{"valid event", validEvent, true, func(bus *mocks.MockEventBus) {
			bus.On("Publish", mock.Anything, mock.AnythingOfType("events.WithdrawValidatedEvent")).Return(nil)
		}},
		{"invalid event", invalidEvent, false, nil},
		{"invalid accountID (empty)", events.WithdrawRequestedEvent{UserID: validUserID, AccountID: uuid.Nil, Amount: 100.0, Currency: "USD"}, false, nil},
		{"zero amount", events.WithdrawRequestedEvent{UserID: validUserID, AccountID: validAccountID, Amount: 0, Currency: "USD"}, false, nil},
		{"negative amount", events.WithdrawRequestedEvent{UserID: validUserID, AccountID: validAccountID, Amount: -10, Currency: "USD"}, false, nil},
		{"missing currency", events.WithdrawRequestedEvent{UserID: validUserID, AccountID: validAccountID, Amount: 100.0, Currency: ""}, false, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := mocks.NewMockEventBus(t)
			if tc.setupMocks != nil {
				tc.setupMocks(bus)
			}
			handler := WithdrawValidationHandler(bus, slog.New(slog.NewTextHandler(io.Discard, nil)))
			ctx := context.Background()
			handler(ctx, tc.input)
			if tc.expectPub {
				assert.True(t, bus.AssertCalled(t, "Publish", ctx, mock.AnythingOfType("events.WithdrawValidatedEvent")), "should publish WithdrawValidatedEvent")
			} else {
				bus.AssertNotCalled(t, "Publish", ctx, mock.AnythingOfType("events.WithdrawValidatedEvent"))
			}
		})
	}
}
