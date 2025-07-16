package account

import (
	"context"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/stretchr/testify/assert"
)

func TestDepositDomainOpHandler_Scenarios(t *testing.T) {
	tests := []struct {
		name      string
		input     domain.Event
		cancelCtx bool
		expectPub bool
	}{
		{
			name:      "valid PaymentInitiatedEvent",
			input:     accountdomain.PaymentInitiatedEvent{},
			cancelCtx: false,
			expectPub: true,
		},
		{
			name:      "wrong event type (DepositRequestedEvent)",
			input:     accountdomain.DepositRequestedEvent{},
			cancelCtx: false,
			expectPub: false,
		},
		{
			name:      "context canceled",
			input:     accountdomain.PaymentInitiatedEvent{},
			cancelCtx: true,
			expectPub: true, // Handler does not check ctx, so still publishes
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := &mockEventBus{}
			handler := DepositDomainOpHandler(bus)
			ctx := context.Background()
			if tc.cancelCtx {
				c, cancel := context.WithCancel(ctx)
				cancel()
				ctx = c
			}
			handler(ctx, tc.input)
			if tc.expectPub {
				assert.Len(t, bus.published, 1)
				_, ok := bus.published[0].(accountdomain.DepositDomainOpDoneEvent)
				assert.True(t, ok, "should publish DepositDomainOpDoneEvent")
			} else {
				assert.Empty(t, bus.published)
			}
		})
	}
}
