package account

import (
	"context"
	"errors"
	"testing"

	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockTransferDomainOperator struct {
	transferFn func(ctx context.Context, senderUserID, receiverUserID, sourceAccountID, destAccountID string, amount float64, currency string) error
}

func (m *mockTransferDomainOperator) Transfer(ctx context.Context, senderUserID, receiverUserID, sourceAccountID, destAccountID string, amount float64, currency string) error {
	return m.transferFn(ctx, senderUserID, receiverUserID, sourceAccountID, destAccountID, amount, currency)
}

func TestTransferDomainOpHandler_BusinessLogic(t *testing.T) {
	validEvent := accountdomain.TransferValidatedEvent{
		TransferRequestedEvent: accountdomain.TransferRequestedEvent{
			EventID:         uuid.New(),
			SourceAccountID: uuid.New(),
			DestAccountID:   uuid.New(),
			SenderUserID:    uuid.New(),
			ReceiverUserID:  uuid.New(),
			Amount:          100,
			Currency:        "USD",
			Source:          accountdomain.MoneySourceInternal,
			Timestamp:       1234567890,
		},
	}
	tests := []struct {
		name      string
		input     accountdomain.TransferValidatedEvent
		operator  *mockTransferDomainOperator
		expectPub bool
	}{
		{
			name:  "domain op success",
			input: validEvent,
			operator: &mockTransferDomainOperator{
				transferFn: func(ctx context.Context, senderUserID, receiverUserID, sourceAccountID, destAccountID string, amount float64, currency string) error {
					return nil
				},
			},
			expectPub: true,
		},
		{
			name:  "domain op error",
			input: validEvent,
			operator: &mockTransferDomainOperator{
				transferFn: func(ctx context.Context, senderUserID, receiverUserID, sourceAccountID, destAccountID string, amount float64, currency string) error {
					return errors.New("domain op failed")
				},
			},
			expectPub: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := &mockEventBus{}
			handler := TransferDomainOpHandler(bus, tc.operator)
			handler(context.Background(), tc.input)
			if tc.expectPub {
				assert.NotEmpty(t, bus.published)
				_, ok := bus.published[0].(accountdomain.TransferDomainOpDoneEvent)
				assert.True(t, ok)
			} else {
				assert.Empty(t, bus.published)
			}
		})
	}
}
