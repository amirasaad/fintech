package account

import (
	"context"
	"testing"

	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTransferValidationHandler(t *testing.T) {
	valid := accountdomain.TransferRequestedEvent{
		EventID:         uuid.New(),
		SourceAccountID: uuid.New(),
		DestAccountID:   uuid.New(),
		SenderUserID:    uuid.New(),
		ReceiverUserID:  uuid.New(),
		Amount:          100,
		Currency:        "USD",
		Source:          accountdomain.MoneySourceInternal,
		Timestamp:       1234567890,
	}
	invalid := accountdomain.TransferRequestedEvent{}
	tests := []struct {
		name      string
		input     accountdomain.TransferRequestedEvent
		expectPub bool
	}{
		{"valid", valid, true},
		{"invalid", invalid, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := &mockEventBus{}
			handler := TransferValidationHandler(bus)
			handler(context.Background(), tc.input)
			if tc.expectPub {
				assert.NotEmpty(t, bus.published)
				_, ok := bus.published[0].(accountdomain.TransferValidatedEvent)
				assert.True(t, ok)
			} else {
				assert.Empty(t, bus.published)
			}
		})
	}
}
