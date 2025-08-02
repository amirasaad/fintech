package events

import (
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// PaymentInitiated is emitted after payment initiation with a provider (event-driven workflow).
type PaymentInitiated struct {
	FlowEvent
	Amount        money.Money
	TransactionID uuid.UUID
	PaymentID     string
	Status        string
}

func (e PaymentInitiated) Type() string { return "PaymentInitiated" }

// PaymentFailedEvent is emitted when payment fails.
type PaymentFailedEvent struct {
	PaymentInitiated
	Reason string
}

func (e PaymentFailedEvent) Type() string { return "PaymentFailedEvent" }

type PaymentProcessed struct {
	PaymentInitiated
}

// PaymentCompleted is an event for when a payment is completed.
type PaymentCompleted struct {
	PaymentInitiated
}

func (e PaymentCompleted) Type() string { return "PaymentCompleted" }

// PaymentIdPersisted is emitted after the paymentId is persisted to the transaction.
type PaymentIdPersisted struct {
	PaymentInitiated
}

func (e PaymentIdPersisted) Type() string { return "PaymentIdPersisted" }
