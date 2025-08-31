package payment

import (
	"github.com/google/uuid"
)

// PaymentStatus represents the status of a payment.
type PaymentStatus string

const (
	// PaymentPending indicates the payment is still pending.
	PaymentPending PaymentStatus = "pending"
	// PaymentCompleted indicates the payment has completed successfully.
	PaymentCompleted PaymentStatus = "completed"
	// PaymentFailed indicates the payment has failed.
	PaymentFailed PaymentStatus = "failed"
)

// PaymentEventType represents the type of payment event.
type PaymentEventType string

const (
	// EventTypePaymentIntentSucceeded is emitted when a payment intent succeeds.
	EventTypePaymentIntentSucceeded PaymentEventType = "payment_intent.succeeded"
	// EventTypePaymentIntentFailed is emitted when a payment intent fails.
	EventTypePaymentIntentFailed PaymentEventType = "payment_intent.failed"
	// EventTypePayoutPaid is emitted when a payout is paid.
	EventTypePayoutPaid PaymentEventType = "payout.paid"
	// EventTypePayoutFailed is emitted when a payout fails.
	EventTypePayoutFailed PaymentEventType = "payout.failed"
)

// PaymentEvent represents a payment-related event from the payment provider.
type PaymentEvent struct {
	ID            string
	TransactionID uuid.UUID
	Status        PaymentStatus
	Amount        int64
	Currency      string
	UserID        uuid.UUID
	AccountID     uuid.UUID
	Metadata      map[string]string
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

// PayoutDestinationType represents the type of destination for a payout
type PayoutDestinationType string

const (
	// PayoutDestinationBankAccount represents a bank account payout destination
	PayoutDestinationBankAccount PayoutDestinationType = "bank_account"
	// PayoutDestinationExternalWallet represents an external wallet payout destination
	PayoutDestinationExternalWallet PayoutDestinationType = "external_wallet"
)

// PayoutDestination represents the destination for a payout
type PayoutDestination struct {
	Type           PayoutDestinationType
	BankAccount    *BankAccountDetails `json:"bank_account,omitempty"`
	ExternalWallet *string             `json:"external_wallet,omitempty"`
}

// BankAccountDetails contains bank account information for payouts
type BankAccountDetails struct {
	AccountNumber     string `json:"account_number"`
	RoutingNumber     string `json:"routing_number"`
	AccountHolderName string `json:"account_holder_name"`
}

// InitiatePayoutParams holds the parameters for initiating a payout
type InitiatePayoutParams struct {
	UserID            uuid.UUID
	AccountID         uuid.UUID
	PaymentProviderID string
	TransactionID     uuid.UUID
	Amount            int64
	Currency          string
	Destination       PayoutDestination
	Description       string
	Metadata          map[string]string
}

// InitiatePayoutResponse represents the response from initiating a payout
type InitiatePayoutResponse struct {
	PayoutID             string
	PaymentProviderID    string
	Status               PaymentStatus
	Amount               int64
	Currency             string
	FeeAmount            int64
	FeeCurrency          string
	EstimatedArrivalDate int64 // Unix timestamp
}
