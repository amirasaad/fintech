package provider

import (
	"context"
	"sync"
	"time"

	"github.com/amirasaad/fintech/pkg/provider"
)

type mockPayment struct {
	status provider.PaymentStatus
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

// UpdatePaymentStatus updates the payment status in the mock provider.
func (m *MockPaymentProvider) UpdatePaymentStatus(
	ctx context.Context,
	i *provider.UpdatePaymentStatusParams,
) error {
	// TODO implement me
	panic("implement me")
}

// HandleWebhook handles payment webhook events
func (m *MockPaymentProvider) HandleWebhook(
	payload []byte,
	signature string,
) (*provider.PaymentEvent, error) {
	// TODO: implement me
	panic("implement me")
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
	params *provider.InitiatePaymentParams,
) (*provider.InitiatePaymentResponse, error) {
	m.mu.Lock()
	m.payments[params.TransactionID.String()] = &mockPayment{
		status: provider.PaymentPending,
	}
	m.mu.Unlock()
	// Simulate async completion
	go func() {
		time.Sleep(2 * time.Second)
		m.mu.Lock()
		m.payments[params.TransactionID.String()].status = provider.PaymentCompleted
		m.mu.Unlock()
	}()
	return &provider.InitiatePaymentResponse{
		Status: provider.PaymentPending,
	}, nil
}

// GetPaymentStatus returns the current status of a payment.
func (m *MockPaymentProvider) GetPaymentStatus(
	ctx context.Context,
	params *provider.GetPaymentStatusParams,
) (provider.PaymentStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.payments[params.PaymentID]; ok {
		return p.status, nil
	}
	return provider.PaymentFailed, nil
}
