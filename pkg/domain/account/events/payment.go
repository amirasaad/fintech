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

func (e PaymentInitiationEvent) EventType() string { return "PaymentInitiationEvent" }
func (e PaymentCompletedEvent) EventType() string  { return "PaymentCompletedEvent" }
func (e PaymentFailedEvent) EventType() string     { return "PaymentFailedEvent" }
