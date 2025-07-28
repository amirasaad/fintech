package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// PaymentInitiationEventOpt is a function that configures a PaymentInitiationEvent
type PaymentInitiationEventOpt func(*PaymentInitiationEvent)

// WithPaymentTransactionID sets the transaction ID for the PaymentInitiationEvent
func WithPaymentTransactionID(id uuid.UUID) PaymentInitiationEventOpt {
	return func(e *PaymentInitiationEvent) { e.TransactionID = id }
}

// WithPaymentAccount sets the account for the PaymentInitiationEvent
func WithPaymentAccount(account *account.Account) PaymentInitiationEventOpt {
	return func(e *PaymentInitiationEvent) { e.Account = account }
}

// WithPaymentAmount sets the amount for the PaymentInitiationEvent
func WithPaymentAmount(amount money.Money) PaymentInitiationEventOpt {
	return func(e *PaymentInitiationEvent) { e.Amount = amount }
}

// WithPaymentTimestamp sets the timestamp for the PaymentInitiationEvent
func WithPaymentTimestamp(t time.Time) PaymentInitiationEventOpt {
	return func(e *PaymentInitiationEvent) { e.Timestamp = t }
}

// WithFlowEvent sets the FlowEvent from an existing FlowEvent
func WithFlowEvent(flowEvent FlowEvent) PaymentInitiationEventOpt {
	return func(e *PaymentInitiationEvent) {
		e.FlowEvent = flowEvent
	}
}

// PaymentInitiationEvent is emitted after payment initiation with a provider.
type PaymentInitiationEvent struct {
	FlowEvent
	ID            uuid.UUID
	TransactionID uuid.UUID
	Account       *account.Account
	Amount        money.Money
	Timestamp     time.Time
}

// NewPaymentInitiationEvent creates a new PaymentInitiationEvent with the given options
func NewPaymentInitiationEvent(opts ...PaymentInitiationEventOpt) *PaymentInitiationEvent {
	event := &PaymentInitiationEvent{
		ID:            uuid.New(),
		TransactionID: uuid.Nil,
		Timestamp:     time.Now(),
	}

	for _, opt := range opts {
		opt(event)
	}

	return event
}

func (e PaymentInitiationEvent) Type() string        { return "PaymentInitiationEvent" }
func (e PaymentInitiationEvent) FlowData() FlowEvent { return e.FlowEvent }

// PaymentCompletedEvent is emitted when payment is confirmed by the provider.
type PaymentCompletedEvent struct {
	FlowEvent
	ID            uuid.UUID
	PaymentID     string
	CorrelationID uuid.UUID
}

func (e PaymentCompletedEvent) Type() string        { return "PaymentCompletedEvent" }
func (e PaymentCompletedEvent) FlowData() FlowEvent { return e.FlowEvent }

// PaymentFailedEvent is emitted when payment fails.
type PaymentFailedEvent struct {
	FlowEvent
	ID            string
	TransactionID uuid.UUID // propagate TransactionID
	PaymentID     string
	Status        string
	Reason        string
}

func (e PaymentFailedEvent) Type() string        { return "PaymentFailedEvent" }
func (e PaymentFailedEvent) FlowData() FlowEvent { return e.FlowEvent }

// PaymentInitiatedEvent is emitted after payment initiation with a provider (event-driven workflow).
type PaymentInitiatedEvent struct {
	FlowEvent
	ID            string
	TransactionID uuid.UUID
	PaymentID     string
	Status        string
}

// NewPaymentInitiatedEvent creates a new PaymentInitiatedEvent with the given parameters
func NewPaymentInitiatedEvent(flowEvent FlowEvent, id string, transactionID uuid.UUID, paymentID string) *PaymentInitiatedEvent {
	return &PaymentInitiatedEvent{
		FlowEvent:     flowEvent,
		ID:            id,
		TransactionID: transactionID,
		PaymentID:     paymentID,
		Status:        "pending",
	}
}

func (e PaymentInitiatedEvent) Type() string        { return "PaymentInitiatedEvent" }
func (e PaymentInitiatedEvent) FlowData() FlowEvent { return e.FlowEvent }

// PaymentIdPersistedEvent is emitted after the paymentId is persisted to the transaction.
type PaymentIdPersistedEvent struct {
	FlowEvent
	ID            string
	TransactionID uuid.UUID // propagate TransactionID
	PaymentID     string
	Status        string
}

func (e PaymentIdPersistedEvent) Type() string        { return "PaymentIdPersistedEvent" }
func (e PaymentIdPersistedEvent) FlowData() FlowEvent { return e.FlowEvent }

// PaymentCompletedEventOpt is a function that configures a PaymentCompletedEvent
type PaymentCompletedEventOpt func(*PaymentCompletedEvent)

// WithCorrelationID sets the correlation ID for the PaymentCompletedEvent
func WithCorrelationID(correlationID uuid.UUID) PaymentCompletedEventOpt {
	return func(e *PaymentCompletedEvent) { e.CorrelationID = correlationID }
}

// NewPaymentCompletedEvent creates a new PaymentCompletedEvent with the given options
func NewPaymentCompletedEvent(
	userID uuid.UUID,
	accountID uuid.UUID,
	opts ...PaymentCompletedEventOpt,
) *PaymentCompletedEvent {
	event := &PaymentCompletedEvent{
		FlowEvent: FlowEvent{
			FlowType:  "payment",
			UserID:    userID,
			AccountID: accountID,
		},
		ID:            uuid.New(),
		PaymentID:     "",
		CorrelationID: uuid.Nil,
	}

	for _, opt := range opts {
		opt(event)
	}

	return event
}
