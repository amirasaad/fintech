package account

import (
	"context"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/stretchr/testify/assert"
)

func TestWithdrawValidationHandler(t *testing.T) {
	validEvent := accountdomain.WithdrawRequestedEvent{
		UserID:    "user-1",
		AccountID: "acc-1",
		Amount:    100.0,
		Currency:  "USD",
	}
	invalidEvent := accountdomain.WithdrawRequestedEvent{
		UserID:    "",
		AccountID: "",
		Amount:    -50.0,
		Currency:  "",
	}

	tests := []struct {
		name      string
		input     domain.Event
		expectPub bool
	}{
		{"valid event", validEvent, true},
		{"invalid event", invalidEvent, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := &mockEventBus{}
			handler := WithdrawValidationHandler(bus)
			ctx := context.Background()
			handler(ctx, tc.input)
			if tc.expectPub {
				assert.Len(t, bus.published, 1)
				_, ok := bus.published[0].(accountdomain.WithdrawValidatedEvent)
				assert.True(t, ok, "should publish WithdrawValidatedEvent")
			} else {
				assert.Empty(t, bus.published)
			}
		})
	}
}
