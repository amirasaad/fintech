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
}

// GetPaymentStatusParams holds the parameters for the GetPaymentStatus method.
type GetPaymentStatusParams struct {
	PaymentID string
}

// PaymentProvider is a interface for payment provider
type PaymentProvider interface {
	InitiatePayment(
		ctx context.Context,
		params *InitiatePaymentParams,
	) (*InitiatePaymentResponse, error)
	GetPaymentStatus(
		ctx context.Context,
		params *GetPaymentStatusParams,
	) (PaymentStatus, error)
}
