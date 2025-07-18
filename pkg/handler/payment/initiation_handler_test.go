package payment

import (
	"context"
	"errors"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockPaymentProvider struct {
	initiateFn func(ctx context.Context, userID, accountID uuid.UUID, amount int64, currency string) (string, error)
}

// GetPaymentStatus implements provider.PaymentProvider.
func (m *mockPaymentProvider) GetPaymentStatus(ctx context.Context, paymentID string) (provider.PaymentStatus, error) {
	panic("unimplemented")
}

func (m *mockPaymentProvider) InitiatePayment(ctx context.Context, userID, accountID uuid.UUID, amount int64, currency string) (string, error) {
	return m.initiateFn(ctx, userID, accountID, amount, currency)
}

func TestPaymentInitiationHandler_BusinessLogic(t *testing.T) {
	userID := uuid.New()
	accountID := uuid.New()
	amount := 100.0 // 100$
	amountInCents := int64(amount * 100)
	currency := "USD"
	paymentID := "pay-123"

	tests := []struct {
		name        string
		input       events.DepositPersistedEvent
		provider    *mockPaymentProvider
		expectPub   bool
		expectPayID string
		expectUser  uuid.UUID
		expectAcc   uuid.UUID
		expectAmt   float64
		expectCur   string
		setupMocks  func(bus *mocks.MockEventBus)
	}{
		{
			name: "provider success",
			input: events.DepositPersistedEvent{
				MoneyCreatedEvent: events.MoneyCreatedEvent{
					DepositValidatedEvent: events.DepositValidatedEvent{
						DepositRequestedEvent: events.DepositRequestedEvent{
							EventID:   uuid.New(),
							UserID:    userID,
							AccountID: accountID,
							Amount:    amount,
							Currency:  currency,
						},
						AccountID: accountID,
					},
					Amount:         amountInCents,
					Currency:       currency,
					TargetCurrency: currency,
					TransactionID:  uuid.New(),
					UserID:         userID,
				},
				UserID: userID, // Ensure top-level UserID is set
				// Add other fields if needed
			},
			provider: &mockPaymentProvider{
				initiateFn: func(ctx context.Context, u, a uuid.UUID, amt int64, cur string) (string, error) {
					assert.Equal(t, userID, u)
					assert.Equal(t, accountID, a)
					assert.InEpsilon(t, amountInCents, amt, 0.1)
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
			setupMocks: func(bus *mocks.MockEventBus) {
				bus.On("Publish", mock.Anything, mock.AnythingOfType("events.PaymentInitiatedEvent")).Return(nil)
			},
		},
		{
			name: "provider error",
			input: events.DepositPersistedEvent{
				MoneyCreatedEvent: events.MoneyCreatedEvent{
					DepositValidatedEvent: events.DepositValidatedEvent{
						DepositRequestedEvent: events.DepositRequestedEvent{
							EventID:   uuid.New(),
							UserID:    userID,
							AccountID: accountID,
							Amount:    amount,
							Currency:  currency,
						},
						AccountID: accountID,
					},
					Amount:         amountInCents,
					Currency:       currency,
					TargetCurrency: currency,
					TransactionID:  uuid.New(),
					UserID:         userID,
				},
				UserID: userID, // Ensure top-level UserID is set
			},
			provider: &mockPaymentProvider{
				initiateFn: func(ctx context.Context, u, a uuid.UUID, amt int64, cur string) (string, error) {
					return "", errors.New("provider failed")
				},
			},
			expectPub:   false,
			expectPayID: "",
			setupMocks:  nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := mocks.NewMockEventBus(t)
			if tc.setupMocks != nil {
				tc.setupMocks(bus)
			}
			handler := PaymentInitiationHandler(bus, tc.provider)
			ctx := context.Background()
			handler(ctx, tc.input)
			if tc.expectPub {
				assert.True(t, bus.AssertCalled(t, "Publish", ctx, mock.AnythingOfType("events.PaymentInitiatedEvent")), "should publish PaymentInitiatedEvent")
			} else {
				bus.AssertNotCalled(t, "Publish", ctx, mock.AnythingOfType("events.PaymentInitiatedEvent"))
			}
		})
	}
}
