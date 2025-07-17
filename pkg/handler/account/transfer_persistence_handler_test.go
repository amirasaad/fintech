package account

import (
	"context"
	"errors"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockTransferPersistenceAdapter struct {
	persistFn func(ctx context.Context, event events.TransferDomainOpDoneEvent) error
}

func (m *mockTransferPersistenceAdapter) PersistTransfer(ctx context.Context, event events.TransferDomainOpDoneEvent) error {
	return m.persistFn(ctx, event)
}

func TestTransferPersistenceHandler_BusinessLogic(t *testing.T) {
	validEvent := events.TransferDomainOpDoneEvent{
		TransferValidatedEvent: events.TransferValidatedEvent{
			TransferRequestedEvent: events.TransferRequestedEvent{
				EventID:         uuid.New(),
				SourceAccountID: uuid.New(),
				DestAccountID:   uuid.New(),
				SenderUserID:    uuid.New(),
				ReceiverUserID:  uuid.New(),
				Amount:          100,
				Currency:        "USD",
				Source:          "Internal",
				Timestamp:       1234567890,
			},
		},
	}
	tests := []struct {
		name      string
		input     events.TransferDomainOpDoneEvent
		adapter   *mockTransferPersistenceAdapter
		expectPub bool
	}{
		{
			name:  "persistence success",
			input: validEvent,
			adapter: &mockTransferPersistenceAdapter{
				persistFn: func(ctx context.Context, event events.TransferDomainOpDoneEvent) error {
					return nil
				},
			},
			expectPub: true,
		},
		{
			name:  "persistence error",
			input: validEvent,
			adapter: &mockTransferPersistenceAdapter{
				persistFn: func(ctx context.Context, event events.TransferDomainOpDoneEvent) error {
					return errors.New("persistence failed")
				},
			},
			expectPub: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := &mockEventBus{}
			handler := TransferPersistenceHandler(bus, tc.adapter)
			handler(context.Background(), tc.input)
			if tc.expectPub {
				assert.NotEmpty(t, bus.published)
				_, ok := bus.published[0].(events.TransferPersistedEvent)
				assert.True(t, ok)
			} else {
				assert.Empty(t, bus.published)
			}
		})
	}
}
