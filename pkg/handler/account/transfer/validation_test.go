package transfer

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func TestValidation(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	validAmount, _ := money.New(100, currency.USD)

	baseEvent := events.TransferRequestedEvent{
		FlowEvent: events.FlowEvent{
			FlowType:      "transfer",
			UserID:        uuid.New(),
			AccountID:     uuid.New(),
			CorrelationID: uuid.New(),
		},
		ID:             uuid.New(),
		Amount:         validAmount,
		Source:         "transfer",
		DestAccountID:  uuid.New(),
		ReceiverUserID: uuid.New(),
	}

	testCases := []struct {
		name           string
		event          events.TransferRequestedEvent
		expectEmit     bool
		malleableEvent func(event *events.TransferRequestedEvent)
	}{
		{
			name:       "valid event",
			event:      baseEvent,
			expectEmit: true,
		},
		{
			name:       "invalid event - nil ID",
			event:      baseEvent,
			expectEmit: false,
			malleableEvent: func(e *events.TransferRequestedEvent) {
				e.ID = uuid.Nil
			},
		},
		{
			name:       "invalid event - nil AccountID",
			event:      baseEvent,
			expectEmit: false,
			malleableEvent: func(e *events.TransferRequestedEvent) {
				e.AccountID = uuid.Nil
			},
		},
		{
			name:       "invalid event - zero amount",
			event:      baseEvent,
			expectEmit: false,
			malleableEvent: func(e *events.TransferRequestedEvent) {
				e.Amount, _ = money.New(0, currency.USD)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bus := mocks.NewMockBus(t)
			event := tc.event
			if tc.malleableEvent != nil {
				tc.malleableEvent(&event)
			}

			if tc.expectEmit {
				bus.On("Emit", ctx, mock.AnythingOfType("events.TransferValidatedEvent")).Return(nil).Once()
			}

			handler := Validation(bus, logger)
			err := handler(ctx, event)

			if err == nil {
				bus.AssertExpectations(t)
			}
		})
	}
}
