package account

import (
	"context"
	"errors"
	"testing"

	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/stretchr/testify/assert"
)

type mockPaymentProvider struct {
	initiateFn func(ctx context.Context, userID, accountID string, amount float64, currency string) (string, error)
}

func (m *mockPaymentProvider) InitiatePayment(ctx context.Context, userID, accountID string, amount float64, currency string) (string, error) {
	return m.initiateFn(ctx, userID, accountID, amount, currency)
}

func TestPaymentInitiationHandler_BusinessLogic(t *testing.T) {
	userID := "user-1"
	accountID := "acc-1"
	amount := 100.0
	currency := "USD"
	paymentID := "pay-123"

	tests := []struct {
		name        string
		input       accountdomain.MoneyConvertedEvent
		provider    *mockPaymentProvider
		expectPub   bool
		expectPayID string
		expectUser  string
		expectAcc   string
		expectAmt   float64
		expectCur   string
	}{
		{
			name: "provider success",
			input: accountdomain.MoneyConvertedEvent{
				UserID:    userID,
				AccountID: accountID,
				Amount:    amount,
				Currency:  currency,
			},
			provider: &mockPaymentProvider{
				initiateFn: func(ctx context.Context, u, a string, amt float64, cur string) (string, error) {
					assert.Equal(t, userID, u)
					assert.Equal(t, accountID, a)
					assert.InEpsilon(t, amount, amt, 0.1)
					assert.Equal(t, currency, cur)
					return paymentID, nil
				},
			},
			expectPub:   true,
			expectPayID: paymentID,
			expectUser:  userID,
			expectAcc:   accountID,
			expectAmt:   amount,
			expectCur:   currency,
		},
		{
			name: "provider error",
			input: accountdomain.MoneyConvertedEvent{
				UserID:    userID,
				AccountID: accountID,
				Amount:    amount,
				Currency:  currency,
			},
			provider: &mockPaymentProvider{
				initiateFn: func(ctx context.Context, u, a string, amt float64, cur string) (string, error) {
					return "", errors.New("provider failed")
				},
			},
			expectPub:   false,
			expectPayID: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := &mockEventBus{}
			handler := PaymentInitiationHandler(bus, tc.provider)
			ctx := context.Background()
			handler(ctx, tc.input)
			if tc.expectPub {
				assert.Len(t, bus.published, 1)
				evt, ok := bus.published[0].(accountdomain.PaymentInitiatedEvent)
				assert.True(t, ok, "should publish PaymentInitiatedEvent")
				assert.Equal(t, tc.expectPayID, evt.PaymentID)
				assert.Equal(t, tc.expectUser, evt.UserID)
				assert.Equal(t, tc.expectAcc, evt.AccountID)
				assert.InEpsilon(t, tc.expectAmt, evt.Amount, 0.1)
				assert.Equal(t, tc.expectCur, evt.Currency)
			} else {
				assert.Empty(t, bus.published)
			}
		})
	}
}
