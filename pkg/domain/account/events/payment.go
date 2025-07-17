package events

// PaymentInitiationEvent is emitted after payment initiation with a provider.
type PaymentInitiationEvent struct {
	MoneyCreatedEvent
	PaymentID string
	Status    string // "pending"
}

// PaymentCompletedEvent is emitted when payment is confirmed by the provider.
type PaymentCompletedEvent struct {
	PaymentInitiationEvent
	// Optionally: add provider response, timestamp, etc.
}

// PaymentFailedEvent is emitted when payment fails.
type PaymentFailedEvent struct {
	PaymentInitiationEvent
	Reason string
}

// PaymentInitiatedEvent is emitted after payment initiation with a provider (event-driven workflow).
type PaymentInitiatedEvent struct {
	DepositPersistedEvent
	PaymentID string
	Status    string // e.g., "initiated"
}

// PaymentIdPersistedEvent is emitted after the paymentId is persisted to the transaction.
type PaymentIdPersistedEvent struct {
	PaymentInitiatedEvent
	// Add DB transaction info if needed
}

func (e PaymentInitiationEvent) EventType() string  { return "PaymentInitiationEvent" }
func (e PaymentCompletedEvent) EventType() string   { return "PaymentCompletedEvent" }
func (e PaymentFailedEvent) EventType() string      { return "PaymentFailedEvent" }
func (e PaymentInitiatedEvent) EventType() string   { return "PaymentInitiatedEvent" }
func (e PaymentIdPersistedEvent) EventType() string { return "PaymentIdPersistedEvent" }
