package stripepayment

import (
	"github.com/google/uuid"
)

// StripeConfig contains the configuration for the Stripe payment provider
type Stripe struct {
	SecretKey      string `envconfig:"SECRET_KEY" required:"true"`
	PublishableKey string `envconfig:"PUBLISHABLE_KEY" required:"true"`
	WebhookURL     string `envconfig:"WEBHOOK_URL"`
	WebhookSecret  string `envconfig:"WEBHOOK_SECRET"`
}

// InitiatePayoutParams contains the parameters needed to initiate a payout
type InitiatePayoutParams struct {
	UserID        uuid.UUID
	AccountID     uuid.UUID
	TransactionID string
	Amount        int64
	Currency      string
	Description   string
}

// InitiatePayoutResponse contains the response from initiating a payout
type InitiatePayoutResponse struct {
	Status   PaymentStatus
	PayoutID string
}

// CreateStripeConnectAccountParams contains the
// parameters needed to create a Stripe Connect account
type CreateStripeConnectAccountParams struct {
	UserID      uuid.UUID
	FirstName   string
	LastName    string
	Email       string
	Country     string
	AccountType string // 'express' or 'standard'
	Individual  Individual
}

// CreateStripeConnectAccountResponse contains the response from creating a Stripe Connect account
type CreateStripeConnectAccountResponse struct {
	AccountID string
	URL       string
}

// Individual represents an individual's details for a Stripe account.

type Individual struct {
	FirstName string
	LastName  string
	Email     string
	Phone     string
	SSNLast4  string
	Address   Address
	DOB       DOB
}

// Address represents a physical address.
type Address struct {
	Line1      string
	City       string
	State      string
	PostalCode string
}

// DOB represents a date of birth.
type DOB struct {
	Day   int
	Month int
	Year  int
}

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	// PaymentPending indicates the payment is pending
	PaymentPending PaymentStatus = "pending"
	// PaymentSucceeded indicates the payment was successful
	PaymentSucceeded PaymentStatus = "succeeded"
	// PaymentFailed indicates the payment failed
	PaymentFailed PaymentStatus = "failed"
)
