package mockpayment

import (
	"context"
	"sync"
	"time"

	"github.com/amirasaad/fintech/pkg/provider/payment"
)

type mockPayment struct {
	status payment.PaymentStatus
}

// MockPaymentProvider simulates a payment provider for tests and local development.
//
// Usage:
// - InitiateDeposit/InitiateWithdraw simulate async payment completion after a short delay.
// - GetPaymentStatus can be polled until PaymentCompleted is returned.
// - This is NOT for production use. Real payment providers use webhooks or callbacks.
//
// In tests, the service will poll GetPaymentStatus until completion,
// simulating a real-world async flow.
//
// See pkg/service/account/account.go for example usage.
// For production, use a real provider and event-driven confirmation.
type MockPaymentProvider struct {
	mu       sync.Mutex
	payments map[string]*mockPayment
}

// NewMockPaymentProvider creates a new instance of MockPaymentProvider.
func NewMockPaymentProvider() *MockPaymentProvider {
	return &MockPaymentProvider{
		payments: make(map[string]*mockPayment),
	}
}

// InitiatePayment simulates initiating a deposit payment.
func (m *MockPaymentProvider) InitiatePayment(
	ctx context.Context,
	params *payment.InitiatePaymentParams,
) (*payment.InitiatePaymentResponse, error) {
	m.mu.Lock()
	m.payments[params.TransactionID.String()] = &mockPayment{
		status: payment.PaymentPending,
	}
	m.mu.Unlock()
	// Simulate async completion
	go func() {
		time.Sleep(2 * time.Second)
		m.mu.Lock()
		m.payments[params.TransactionID.String()].status = payment.PaymentCompleted
		m.mu.Unlock()
	}()
	return &payment.InitiatePaymentResponse{
		Status: payment.PaymentPending,
	}, nil
}

// HandleWebhook handles payment webhook events
func (m *MockPaymentProvider) HandleWebhook(
	ctx context.Context,
	payload []byte,
	signature string,
) (*payment.PaymentEvent, error) {
	// TODO: implement me
	panic("implement me")
}
