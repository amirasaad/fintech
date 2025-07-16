package account

import (
	"context"
	"errors"
	"testing"

	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/stretchr/testify/assert"
)

type mockDepositAdapter struct {
	depositFn func(ctx context.Context, userID, accountID string, amount float64, currency string) error
}

func (m *mockDepositAdapter) Deposit(ctx context.Context, userID, accountID string, amount float64, currency string) error {
	return m.depositFn(ctx, userID, accountID, amount, currency)
}

func TestDepositDomainOpHandler_BusinessLogic(t *testing.T) {
	userID := "user-1"
	accountID := "acc-1"
	amount := 100.0
	currency := "USD"

	tests := []struct {
		name       string
		input      accountdomain.PaymentInitiatedEvent
		s          DepositDomainOperator
		expectPub  bool
		expectUser string
		expectAcc  string
		expectAmt  float64
		expectCur  string
	}{
		{
			name: "domain op success",
			input: accountdomain.PaymentInitiatedEvent{
				MoneyConvertedEvent: accountdomain.MoneyConvertedEvent{
					UserID:    userID,
					AccountID: accountID,
					Amount:    amount,
					Currency:  currency,
				},
			},
			s: &mockDepositAdapter{
				depositFn: func(ctx context.Context, u, a string, amt float64, cur string) error {
					assert.Equal(t, userID, u)
					assert.Equal(t, accountID, a)
					assert.InEpsilon(t, amount, amt, 0.1)
					assert.Equal(t, currency, cur)
					return nil
				},
			},
			expectPub:  true,
			expectUser: userID,
			expectAcc:  accountID,
			expectAmt:  amount,
			expectCur:  currency,
		},
		{
			name: "domain op error",
			input: accountdomain.PaymentInitiatedEvent{
				MoneyConvertedEvent: accountdomain.MoneyConvertedEvent{
					UserID:    userID,
					AccountID: accountID,
					Amount:    amount,
					Currency:  currency,
				},
			},
			s: &mockDepositAdapter{
				depositFn: func(ctx context.Context, u, a string, amt float64, cur string) error {
					return errors.New("domain op failed")
				},
			},
			expectPub: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := &mockEventBus{}
			handler := DepositDomainOpHandler(bus, tc.s)
			ctx := context.Background()
			handler(ctx, tc.input)
			if tc.expectPub {
				assert.Len(t, bus.published, 1)
				evt, ok := bus.published[0].(accountdomain.DepositDomainOpDoneEvent)
				assert.True(t, ok, "should publish DepositDomainOpDoneEvent")
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
