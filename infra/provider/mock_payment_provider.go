package provider

import (
	"context"
	"sync"
	"time"

	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/google/uuid"
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
// In tests, the service will poll GetPaymentStatus until completion, simulating a real-world async flow.
//
// See pkg/service/account/account.go for example usage.
//
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
func (m *MockPaymentProvider) InitiatePayment(ctx context.Context, userID, accountID uuid.UUID, amount int64, currency string) (string, error) {
	paymentID := uuid.New().String()
	m.mu.Lock()
	m.payments[paymentID] = &mockPayment{status: provider.PaymentPending}
	m.mu.Unlock()
	// Simulate async completion
	go func() {
		time.Sleep(2 * time.Second)
		m.mu.Lock()
		m.payments[paymentID].status = provider.PaymentCompleted
		m.mu.Unlock()
	}()
	return paymentID, nil
}

// GetPaymentStatus returns the current status of a payment.
func (m *MockPaymentProvider) GetPaymentStatus(ctx context.Context, paymentID string) (provider.PaymentStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.payments[paymentID]; ok {
		return p.status, nil
	}
	return provider.PaymentFailed, nil
}
