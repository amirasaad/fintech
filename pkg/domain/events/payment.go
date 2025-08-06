package events

import (
	"github.com/amirasaad/fintech/pkg/domain/account"
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

func (e PaymentInitiated) Type() string { return EventTypePaymentInitiated.String() }

// PaymentFailed is emitted when payment fails.
type PaymentFailed struct {
	PaymentInitiated
	Reason string
}

func (e PaymentFailed) Type() string { return EventTypePaymentFailed.String() }

type PaymentProcessed struct {
	PaymentInitiated
}

func (e PaymentProcessed) Type() string { return EventTypePaymentProcessed.String() }

// PaymentCompleted is an event for when a payment is completed.
type PaymentCompleted struct {
	PaymentInitiated
	ProviderFee account.Fee
}

func (e PaymentCompleted) Type() string { return EventTypePaymentCompleted.String() }
