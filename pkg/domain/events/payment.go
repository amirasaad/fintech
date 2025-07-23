package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// PaymentInitiationEvent is emitted after payment initiation with a provider.
type PaymentInitiationEvent struct {
	FlowEvent
	ID            uuid.UUID
	TransactionID uuid.UUID
	Account       *account.Account
	Amount        money.Money
	Timestamp     time.Time
}

// PaymentCompletedEvent is emitted when payment is confirmed by the provider.
type PaymentCompletedEvent struct {
	ID            uuid.UUID
	PaymentID     string
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
