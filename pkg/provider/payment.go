package provider

import (
	"context"

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

// PaymentProvider is a interface for payment provider
type PaymentProvider interface {
	InitiatePayment(ctx context.Context, userID, accountID uuid.UUID, amount float64, currency string) (string, error)
	GetPaymentStatus(ctx context.Context, paymentID string) (PaymentStatus, error)
}
