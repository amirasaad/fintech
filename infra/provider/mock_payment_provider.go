package provider

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PaymentStatus represents the status of a payment in the mock provider.
type PaymentStatus string

const (
	// PaymentPending indicates the payment is still pending.
	PaymentPending PaymentStatus = "pending"
	// PaymentCompleted indicates the payment has completed successfully.
	PaymentCompleted PaymentStatus = "completed"
	// PaymentFailed indicates the payment has failed.
	PaymentFailed PaymentStatus = "failed"
)

type mockPayment struct {
	status PaymentStatus
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

// InitiateDeposit simulates initiating a deposit payment.
func (m *MockPaymentProvider) InitiateDeposit(ctx context.Context, userID, accountID uuid.UUID, amount float64, currency string) (string, error) {
	paymentID := uuid.New().String()
	m.mu.Lock()
	m.payments[paymentID] = &mockPayment{status: PaymentPending}
	m.mu.Unlock()
	// Simulate async completion
	go func() {
		time.Sleep(2 * time.Second)
		m.mu.Lock()
		m.payments[paymentID].status = PaymentCompleted
		m.mu.Unlock()
	}()
	return paymentID, nil
}

// InitiateWithdraw simulates initiating a withdrawal payment.
func (m *MockPaymentProvider) InitiateWithdraw(ctx context.Context, userID, accountID uuid.UUID, amount float64, currency, externalTarget string) (string, error) {
	paymentID := uuid.New().String()
	m.mu.Lock()
	m.payments[paymentID] = &mockPayment{status: PaymentPending}
	m.mu.Unlock()
	// Simulate async completion
	go func() {
		time.Sleep(2 * time.Second)
		m.mu.Lock()
		m.payments[paymentID].status = PaymentCompleted
		m.mu.Unlock()
	}()
	return paymentID, nil
}

// GetPaymentStatus returns the current status of a payment.
func (m *MockPaymentProvider) GetPaymentStatus(ctx context.Context, paymentID string) (PaymentStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.payments[paymentID]; ok {
		return p.status, nil
	}
	return PaymentFailed, nil
}
