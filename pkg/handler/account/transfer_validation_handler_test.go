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

func TestTransferValidationHandler(t *testing.T) {
	valid := events.TransferRequestedEvent{
		EventID:         uuid.New(),
		SourceAccountID: uuid.New(),
		DestAccountID:   uuid.New(),
		SenderUserID:    uuid.New(),
		ReceiverUserID:  uuid.New(),
		Amount:          100,
		Currency:        "USD",
		Source:          "Internal",
		Timestamp:       1234567890,
	}
	invalid := events.TransferRequestedEvent{}
	tests := []struct {
		name      string
		input     events.TransferRequestedEvent
		expectPub bool
	}{
		{"valid", valid, true},
		{"invalid", invalid, false},
		{"invalid sender UUID", func() events.TransferRequestedEvent { e := valid; e.SenderUserID = uuid.Nil; return e }(), false},
		{"invalid source account UUID", func() events.TransferRequestedEvent { e := valid; e.SourceAccountID = uuid.Nil; return e }(), false},
		{"invalid dest account UUID", func() events.TransferRequestedEvent { e := valid; e.DestAccountID = uuid.Nil; return e }(), false},
		{"zero amount", func() events.TransferRequestedEvent { e := valid; e.Amount = 0; return e }(), false},
		{"negative amount", func() events.TransferRequestedEvent { e := valid; e.Amount = -10; return e }(), false},
		{"missing currency", func() events.TransferRequestedEvent { e := valid; e.Currency = ""; return e }(), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := &mockEventBus{}
			handler := TransferValidationHandler(bus, slog.New(slog.NewTextHandler(io.Discard, nil)))
			handler(context.Background(), tc.input)
			if tc.expectPub {
				assert.NotEmpty(t, bus.published)
				_, ok := bus.published[0].(events.TransferValidatedEvent)
				assert.True(t, ok)
			} else {
				assert.Empty(t, bus.published)
			}
		})
	}
}
