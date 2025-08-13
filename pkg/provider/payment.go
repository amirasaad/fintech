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

// PaymentEvent represents a payment event from Stripe.
type PaymentEvent struct {
	ID        string
	Status    PaymentStatus
	Amount    int64
	Currency  string
	UserID    uuid.UUID
	AccountID uuid.UUID
	Metadata  map[string]string
}

// InitiatePaymentParams holds the parameters for the InitiatePayment method.
type InitiatePaymentParams struct {
	UserID        uuid.UUID
	AccountID     uuid.UUID
	TransactionID uuid.UUID
	Amount        int64
	Currency      string
}

type InitiatePaymentResponse struct {
	Status PaymentStatus
	// PaymentID is the ID of the payment in the payment provider
	// (e.g., Stripe Checkout Session ID)
	PaymentID string
}

// GetPaymentStatusParams holds the parameters for the GetPaymentStatus method.
type GetPaymentStatusParams struct {
	PaymentID string
}

// UpdatePaymentStatusParams holds the parameters for updating a payment status
type UpdatePaymentStatusParams struct {
	TransactionID uuid.UUID
	PaymentID     string
	Status        PaymentStatus
}

// Payment is a interface for payment provider
type Payment interface {
	InitiatePayment(
		ctx context.Context,
		params *InitiatePaymentParams,
	) (*InitiatePaymentResponse, error)
	HandleWebhook(
		ctx context.Context,
		payload []byte,
		signature string,
	) (*PaymentEvent, error)
}
