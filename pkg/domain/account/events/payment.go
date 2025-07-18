package events

import "github.com/google/uuid"

// PaymentInitiationEvent is emitted after payment initiation with a provider.
type PaymentInitiationEvent struct {
	MoneyCreatedEvent
	PaymentID     string
	Status        string    // "pending"
	TransactionID uuid.UUID // propagate TransactionID
	UserID        string    // propagate UserID
}

// PaymentCompletedEvent is emitted when payment is confirmed by the provider.
type PaymentCompletedEvent struct {
	PaymentInitiationEvent
	// Optionally: add provider response, timestamp, etc.
	TransactionID uuid.UUID // propagate TransactionID
	UserID        uuid.UUID // propagate UserID
}

// PaymentFailedEvent is emitted when payment fails.
type PaymentFailedEvent struct {
	PaymentInitiationEvent
	Reason        string
	TransactionID uuid.UUID // propagate TransactionID
	UserID        uuid.UUID // propagate UserID
}

// PaymentInitiatedEvent is emitted after payment initiation with a provider (event-driven workflow).
type PaymentInitiatedEvent struct {
	DepositPersistedEvent
	PaymentID     string
	Status        string    // e.g., "initiated"
	TransactionID uuid.UUID // propagate TransactionID
	UserID        uuid.UUID // propagate UserID
}

// PaymentIdPersistedEvent is emitted after the paymentId is persisted to the transaction.
type PaymentIdPersistedEvent struct {
	PaymentInitiatedEvent
	TransactionID uuid.UUID // propagate TransactionID
	UserID        uuid.UUID // propagate UserID
	// Add DB transaction info if needed
}

func (e PaymentInitiationEvent) EventType() string  { return "PaymentInitiationEvent" }
func (e PaymentCompletedEvent) EventType() string   { return "PaymentCompletedEvent" }
func (e PaymentFailedEvent) EventType() string      { return "PaymentFailedEvent" }
func (e PaymentInitiatedEvent) EventType() string   { return "PaymentInitiatedEvent" }
func (e PaymentIdPersistedEvent) EventType() string { return "PaymentIdPersistedEvent" }
