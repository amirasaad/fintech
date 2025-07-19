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
	senderID := uuid.New()
	sourceAccountID := uuid.New()
	destAccountID := uuid.New()
	receiverID := uuid.New()

	valid := events.TransferRequestedEvent{
		FlowEvent: events.FlowEvent{
			FlowType:      "transfer",
			UserID:        senderID,
			AccountID:     sourceAccountID,
			CorrelationID: uuid.New(),
		},
		ID:             uuid.New(),
		Amount:         money.NewFromData(1000, "USD"),
		Source:         "transfer",
		DestAccountID:  destAccountID,
		ReceiverUserID: receiverID,
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
		{"invalid sender UUID", func() events.TransferRequestedEvent { e := valid; e.UserID = uuid.Nil; return e }(), false, func(bus *mocks.MockEventBus) {
			bus.On("Publish", mock.Anything, mock.AnythingOfType("events.TransferValidatedEvent")).Return(nil)
		}},
		{"invalid source account UUID", func() events.TransferRequestedEvent { e := valid; e.AccountID = uuid.Nil; return e }(), false, func(bus *mocks.MockEventBus) {
			bus.On("Publish", mock.Anything, mock.AnythingOfType("events.TransferValidatedEvent")).Return(nil)
		}},
		{"invalid dest account UUID", func() events.TransferRequestedEvent { e := valid; e.DestAccountID = uuid.Nil; return e }(), false, nil},
		{"zero amount", func() events.TransferRequestedEvent { e := valid; e.Amount = money.NewFromData(0, "USD"); return e }(), false, nil},
		{"negative amount", func() events.TransferRequestedEvent { e := valid; e.Amount = money.NewFromData(-1000, "USD"); return e }(), false, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := mocks.NewMockEventBus(t)
			if tc.expectPub && tc.setupMocks != nil {
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
