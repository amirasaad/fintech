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

func TestWithdrawValidationHandler(t *testing.T) {
	validUserID := uuid.NewString()
	validAccountID := uuid.NewString()
	validEvent := events.WithdrawRequestedEvent{
		UserID:    validUserID,
		AccountID: validAccountID,
		Amount:    100.0,
		Currency:  "USD",
	}
	invalidEvent := events.WithdrawRequestedEvent{
		UserID:    "",
		AccountID: "",
		Amount:    -50.0,
		Currency:  "",
	}

	tests := []struct {
		name      string
		input     events.WithdrawRequestedEvent
		expectPub bool
	}{
		{"valid event", validEvent, true},
		{"invalid event", invalidEvent, false},
		{"invalid userID (not UUID)", events.WithdrawRequestedEvent{UserID: "not-a-uuid", AccountID: validAccountID, Amount: 100.0, Currency: "USD"}, false},
		{"invalid accountID (empty)", events.WithdrawRequestedEvent{UserID: validUserID, AccountID: "", Amount: 100.0, Currency: "USD"}, false},
		{"zero amount", events.WithdrawRequestedEvent{UserID: validUserID, AccountID: validAccountID, Amount: 0, Currency: "USD"}, false},
		{"negative amount", events.WithdrawRequestedEvent{UserID: validUserID, AccountID: validAccountID, Amount: -10, Currency: "USD"}, false},
		{"missing currency", events.WithdrawRequestedEvent{UserID: validUserID, AccountID: validAccountID, Amount: 100.0, Currency: ""}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := &mockEventBus{}
			handler := WithdrawValidationHandler(bus, slog.New(slog.NewTextHandler(io.Discard, nil)))
			ctx := context.Background()
			handler(ctx, tc.input)
			if tc.expectPub {
				assert.Len(t, bus.published, 1)
				_, ok := bus.published[0].(events.WithdrawValidatedEvent)
				assert.True(t, ok, "should publish WithdrawValidatedEvent")
			} else {
				assert.Empty(t, bus.published)
			}
		})
	}
}
