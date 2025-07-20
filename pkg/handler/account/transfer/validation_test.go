package transfer

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

type mockBus struct {
	handlers map[string][]eventbus.HandlerFunc
}

func (m *mockBus) Emit(ctx context.Context, event domain.Event) error {
	handlers := m.handlers[event.Type()]
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockBus) Register(eventType string, handler eventbus.HandlerFunc) {
	if m.handlers == nil {
		m.handlers = make(map[string][]eventbus.HandlerFunc)
	}
	m.handlers[eventType] = append(m.handlers[eventType], handler)
}

func TestTransferValidation(t *testing.T) {
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
		setupMocks func(bus *mockBus)
	}{
		{"valid", valid, true, func(bus *mockBus) {
			bus.Register("TransferValidatedEvent", func(ctx context.Context, event domain.Event) error {
				assert.Equal(t, "TransferValidatedEvent", event.Type())
				return nil
			})
		}},
		{"invalid", invalid, false, nil},
		{"invalid sender UUID", func() events.TransferRequestedEvent { e := valid; e.UserID = uuid.Nil; return e }(), false, func(bus *mockBus) {
			bus.Register("TransferValidatedEvent", func(ctx context.Context, event domain.Event) error {
				assert.Equal(t, "TransferValidatedEvent", event.Type())
				return nil
			})
		}},
		{"invalid source account UUID", func() events.TransferRequestedEvent { e := valid; e.AccountID = uuid.Nil; return e }(), false, func(bus *mockBus) {
			bus.Register("TransferValidatedEvent", func(ctx context.Context, event domain.Event) error {
				assert.Equal(t, "TransferValidatedEvent", event.Type())
				return nil
			})
		}},
		{"invalid dest account UUID", func() events.TransferRequestedEvent { e := valid; e.DestAccountID = uuid.Nil; return e }(), false, nil},
		{"zero amount", func() events.TransferRequestedEvent { e := valid; e.Amount = money.NewFromData(0, "USD"); return e }(), false, nil},
		{"negative amount", func() events.TransferRequestedEvent { e := valid; e.Amount = money.NewFromData(-1000, "USD"); return e }(), false, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := &mockBus{}
			if tc.expectPub && tc.setupMocks != nil {
				tc.setupMocks(bus)
			}
			handler := Validation(bus, slog.New(slog.NewTextHandler(io.Discard, nil)))
			handler(context.Background(), tc.input) //nolint:errcheck
			if tc.expectPub {
				assert.True(t, bus.handlers["TransferValidatedEvent"] != nil, "should publish TransferValidatedEvent")
			} else {
				assert.True(t, bus.handlers["TransferValidatedEvent"] == nil, "should not publish TransferValidatedEvent")
			}
		})
	}
}
