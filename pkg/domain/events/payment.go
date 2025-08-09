package events

import (
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/money"
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

func (e *PaymentInitiated) WithAmount(m money.Money) *PaymentInitiated {
	e.Amount = m
	return e
}

func (e *PaymentInitiated) WithTransactionID(id uuid.UUID) *PaymentInitiated {
	e.TransactionID = id
	return e
}

func (e *PaymentInitiated) WithPaymentID(id string) *PaymentInitiated {
	e.PaymentID = id
	return e
}

func (e *PaymentInitiated) WithStatus(status string) *PaymentInitiated {
	e.Status = status
	return e
}

// PaymentFailed is emitted when payment fails.
type PaymentFailed struct {
	PaymentInitiated
	Reason string
}

func (e *PaymentFailed) Type() string { return EventTypePaymentFailed.String() }

func (e *PaymentFailed) WithReason(reason string) *PaymentFailed {
	e.Reason = reason
	return e
}

type PaymentProcessed struct {
	PaymentInitiated
}

func (e *PaymentProcessed) Type() string { return EventTypePaymentProcessed.String() }

func (e *PaymentProcessed) WithAmount(m money.Money) *PaymentProcessed {
	e.Amount = m
	return e
}

// PaymentCompleted is an event for when a payment is completed.
type PaymentCompleted struct {
	PaymentInitiated
	ProviderFee account.Fee
}

func (e PaymentCompleted) Type() string { return EventTypePaymentCompleted.String() }
