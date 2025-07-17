package deposit

import (
	"context"
	"errors"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
		input      events.PaymentInitiatedEvent
		s          DepositDomainOperator
		expectPub  bool
		expectUser string
		expectAcc  string
		expectAmt  float64
		expectCur  string
		setupMocks func(bus *mocks.MockEventBus)
	}{
		{
			name: "domain op success",
			input: events.PaymentInitiatedEvent{
				DepositPersistedEvent: events.DepositPersistedEvent{
					MoneyCreatedEvent: events.MoneyCreatedEvent{
						DepositValidatedEvent: events.DepositValidatedEvent{
							DepositRequestedEvent: events.DepositRequestedEvent{
								AccountID: accountID,
								UserID:    userID,
								Amount:    amount,
								Currency:  currency,
							},
							AccountID: accountID,
						},
						Amount:   int64(amount),
						Currency: currency,
					},
				},
				PaymentID: "payment-1",
				Status:    "initiated",
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
			setupMocks: func(bus *mocks.MockEventBus) {
				bus.On("Publish", mock.Anything, mock.AnythingOfType("events.DepositPersistedEvent")).Return(nil)
			},
		},
		{
			name: "domain op error",
			input: events.PaymentInitiatedEvent{
				DepositPersistedEvent: events.DepositPersistedEvent{
					MoneyCreatedEvent: events.MoneyCreatedEvent{
						DepositValidatedEvent: events.DepositValidatedEvent{
							DepositRequestedEvent: events.DepositRequestedEvent{
								AccountID: accountID,
								UserID:    userID,
								Amount:    amount,
								Currency:  currency,
							},
							AccountID: accountID,
						},
						Amount:   int64(amount),
						Currency: currency,
					},
				},
				PaymentID: "payment-1",
				Status:    "initiated",
			},
			s: &mockDepositAdapter{
				depositFn: func(ctx context.Context, u, a string, amt float64, cur string) error {
					return errors.New("domain op failed")
				},
			},
			expectPub:  false,
			setupMocks: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := mocks.NewMockEventBus(t)
			handler := DepositDomainOpHandler(bus, tc.s)
			ctx := context.Background()
			if tc.setupMocks != nil {
				tc.setupMocks(bus)
			}
			handler(ctx, tc.input)
			if tc.expectPub {
				assert.True(t, bus.AssertCalled(t, "Publish", mock.Anything, mock.AnythingOfType("events.DepositPersistedEvent")), "should publish DepositPersistedEvent")
			} else {
				bus.AssertNotCalled(t, "Publish", mock.Anything, mock.AnythingOfType("events.DepositPersistedEvent"))
			}
		})
	}
}
