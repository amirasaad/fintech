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

// PaymentFailed is emitted when payment fails.
type PaymentFailed struct {
	PaymentInitiated
	Reason string
}

func (e PaymentFailed) Type() string { return "PaymentFailed" }

type PaymentProcessed struct {
	PaymentInitiated
}

// PaymentCompleted is an event for when a payment is completed.
type PaymentCompleted struct {
	PaymentInitiated
}

func (e PaymentCompleted) Type() string { return "PaymentCompleted" }

// PaymentPersisted is emitted after the payment ID is persisted to the transaction.
type PaymentPersisted struct {
	PaymentInitiated
}

func (e PaymentPersisted) Type() string { return "PaymentPersisted" }
