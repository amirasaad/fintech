package transfer

import (
	"context"
	"errors"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
		name       string
		input      events.TransferDomainOpDoneEvent
		adapter    *mockTransferPersistenceAdapter
		expectPub  bool
		setupMocks func(bus *mocks.MockEventBus)
	}{
		{
			name:      "persistence success",
			input:     validEvent,
			adapter:   &mockTransferPersistenceAdapter{persistFn: func(ctx context.Context, event events.TransferDomainOpDoneEvent) error { return nil }},
			expectPub: true,
			setupMocks: func(bus *mocks.MockEventBus) {
				bus.On("Publish", mock.Anything, mock.AnythingOfType("events.TransferPersistedEvent")).Return(nil)
			},
		},
		{
			name:  "persistence error",
			input: validEvent,
			adapter: &mockTransferPersistenceAdapter{persistFn: func(ctx context.Context, event events.TransferDomainOpDoneEvent) error {
				return errors.New("persistence failed")
			}},
			expectPub:  false,
			setupMocks: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := mocks.NewMockEventBus(t)
			if tc.setupMocks != nil {
				tc.setupMocks(bus)
			}
			handler := TransferPersistenceHandler(bus, tc.adapter)
			handler(context.Background(), tc.input)
			if tc.expectPub {
				assert.True(t, bus.AssertCalled(t, "Publish", mock.Anything, mock.AnythingOfType("events.TransferPersistedEvent")), "should publish TransferPersistedEvent")
			} else {
				bus.AssertNotCalled(t, "Publish", mock.Anything, mock.AnythingOfType("events.TransferPersistedEvent"))
			}
		})
	}
}
