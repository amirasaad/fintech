package transfer

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTransferValidationHandler(t *testing.T) {
	valid := events.TransferRequestedEvent{
		EventID:         uuid.New(),
		SourceAccountID: uuid.New(),
		DestAccountID:   uuid.New(),
		SenderUserID:    uuid.New(),
		ReceiverUserID:  uuid.New(),
		Amount:          money.NewFromData(10000, "USD"),
		Source:          "Internal",
		Timestamp:       1234567890,
	}
	invalid := events.TransferRequestedEvent{}
	tests := []struct {
		name       string
		input      events.TransferRequestedEvent
		expectPub  bool
		setupMocks func(bus *mocks.MockEventBus)
	}{
		{"valid", valid, true, func(bus *mocks.MockEventBus) {
			bus.On("Publish", mock.Anything, mock.AnythingOfType("events.TransferValidatedEvent")).Return(nil)
		}},
		{"invalid", invalid, false, nil},
		{"invalid sender UUID", func() events.TransferRequestedEvent { e := valid; e.SenderUserID = uuid.Nil; return e }(), false, nil},
		{"invalid source account UUID", func() events.TransferRequestedEvent { e := valid; e.SourceAccountID = uuid.Nil; return e }(), false, nil},
		{"invalid dest account UUID", func() events.TransferRequestedEvent { e := valid; e.DestAccountID = uuid.Nil; return e }(), false, nil},
		{"zero amount", func() events.TransferRequestedEvent { e := valid; e.Amount = money.NewFromData(0, "USD"); return e }(), false, nil},
		{"negative amount", func() events.TransferRequestedEvent { e := valid; e.Amount = money.NewFromData(-1000, "USD"); return e }(), false, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := mocks.NewMockEventBus(t)
			if tc.setupMocks != nil {
				tc.setupMocks(bus)
			}
			handler := TransferValidationHandler(bus, slog.New(slog.NewTextHandler(io.Discard, nil)))
			handler(context.Background(), tc.input)
			if tc.expectPub {
				assert.True(t, bus.AssertCalled(t, "Publish", mock.Anything, mock.AnythingOfType("events.TransferValidatedEvent")), "should publish TransferValidatedEvent")
			} else {
				bus.AssertNotCalled(t, "Publish", mock.Anything, mock.AnythingOfType("events.TransferValidatedEvent"))
			}
		})
	}
}
