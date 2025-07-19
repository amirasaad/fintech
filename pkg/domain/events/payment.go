package events

import "github.com/google/uuid"

// PaymentInitiationEvent is emitted after payment initiation with a provider.
type PaymentInitiationEvent struct {
	ID            string
	Status        string    // "pending"
	TransactionID uuid.UUID // propagate TransactionID
}

// PaymentCompletedEvent is emitted when payment is confirmed by the provider.
type PaymentCompletedEvent struct {
	ID            string
	TransactionID uuid.UUID // propagate TransactionID
	PaymentID     string
	Status        string
	UserID        uuid.UUID
	AccountID     uuid.UUID
	CorrelationID uuid.UUID
}

// PaymentFailedEvent is emitted when payment fails.
type PaymentFailedEvent struct {
	ID            string
	TransactionID uuid.UUID // propagate TransactionID
	PaymentID     string
	Status        string
	Reason        string
	UserID        uuid.UUID
	AccountID     uuid.UUID
	CorrelationID uuid.UUID
}

// PaymentInitiatedEvent is emitted after payment initiation with a provider (event-driven workflow).
type PaymentInitiatedEvent struct {
	ID            string
	TransactionID uuid.UUID
	PaymentID     string
	Status        string
	UserID        uuid.UUID
	AccountID     uuid.UUID
	CorrelationID uuid.UUID
}

// PaymentIdPersistedEvent is emitted after the paymentId is persisted to the transaction.
type PaymentIdPersistedEvent struct {
	ID            string
	TransactionID uuid.UUID // propagate TransactionID
	PaymentID     string
	Status        string
	UserID        uuid.UUID
	AccountID     uuid.UUID
	CorrelationID uuid.UUID
}

func (e PaymentInitiationEvent) Type() string  { return "PaymentInitiationEvent" }
func (e PaymentCompletedEvent) Type() string   { return "PaymentCompletedEvent" }
func (e PaymentFailedEvent) Type() string      { return "PaymentFailedEvent" }
func (e PaymentInitiatedEvent) Type() string   { return "PaymentInitiatedEvent" }
func (e PaymentIdPersistedEvent) Type() string { return "PaymentIdPersistedEvent" }
