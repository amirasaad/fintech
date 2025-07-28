package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// DepositPersistedEventOpt is a function that configures a DepositPersistedEvent
type DepositPersistedEventOpt func(*DepositPersistedEvent)

// --- DepositRequestedEvent ---
type DepositRequestedEventOpt func(*DepositRequestedEvent)

func WithDepositAmount(m money.Money) DepositRequestedEventOpt {
	return func(e *DepositRequestedEvent) { e.Amount = m }
}
func WithDepositTimestamp(ts time.Time) DepositRequestedEventOpt {
	return func(e *DepositRequestedEvent) { e.Timestamp = ts }
}
func WithDepositID(id uuid.UUID) DepositRequestedEventOpt {
	return func(e *DepositRequestedEvent) { e.ID = id }
}
func WithDepositFlowEvent(fe FlowEvent) DepositRequestedEventOpt {
	return func(e *DepositRequestedEvent) { e.FlowEvent = fe }
}

func NewDepositRequestedEvent(userID, accountID, correlationID uuid.UUID, opts ...DepositRequestedEventOpt) *DepositRequestedEvent {
	event := DepositRequestedEvent{
		FlowEvent: FlowEvent{
			FlowType:      "deposit",
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: correlationID,
		},
		ID:        uuid.New(),
		Amount:    money.Zero(currency.USD),
		Timestamp: time.Now(),
	}
	for _, opt := range opts {
		opt(&event)
	}
	return &event
}

// --- DepositValidatedEvent ---
type DepositValidatedEventOpt func(*DepositValidatedEvent)

func WithDepositRequestedEvent(e DepositRequestedEvent) DepositValidatedEventOpt {
	return func(v *DepositValidatedEvent) { v.DepositRequestedEvent = e }
}
func NewDepositValidatedEvent(userID, accountID, correlationID uuid.UUID, opts ...DepositValidatedEventOpt) *DepositValidatedEvent {
	// By default, embed a valid DepositRequestedEvent
	dre := NewDepositRequestedEvent(userID, accountID, correlationID)
	v := DepositValidatedEvent{
		DepositRequestedEvent: *dre,
	}
	for _, opt := range opts {
		opt(&v)
	}
	return &v
}

// --- DepositBusinessValidationEvent ---
type DepositBusinessValidationEventOpt func(*DepositBusinessValidationEvent)

func WithDepositValidatedEvent(e DepositValidatedEvent) DepositBusinessValidationEventOpt {
	return func(bv *DepositBusinessValidationEvent) { bv.DepositValidatedEvent = e }
}
func WithBusinessValidationAmount(m money.Money) DepositBusinessValidationEventOpt {
	return func(bv *DepositBusinessValidationEvent) { bv.Amount = m }
}
func NewDepositBusinessValidationEvent(userID, accountID, correlationID uuid.UUID, opts ...DepositBusinessValidationEventOpt) *DepositBusinessValidationEvent {
	// By default, embed a valid DepositValidatedEvent
	ve := NewDepositValidatedEvent(userID, accountID, correlationID)
	bv := DepositBusinessValidationEvent{
		DepositValidatedEvent: *ve,
		Amount:                money.Zero(currency.USD),
	}
	for _, opt := range opts {
		opt(&bv)
	}
	return &bv
}

// --- DepositFailedEvent ---
type DepositFailedEventOpt func(*DepositFailedEvent)

func WithFailureReason(reason string) DepositFailedEventOpt {
	return func(df *DepositFailedEvent) { df.Reason = reason }
}

// WithDepositFailedTransactionID sets the transaction ID for a failed deposit event
func WithDepositFailedTransactionID(id uuid.UUID) DepositFailedEventOpt {
	return func(df *DepositFailedEvent) { df.TransactionID = id }
}

func NewDepositFailedEvent(flowEvent FlowEvent, reason string, opts ...DepositFailedEventOpt) *DepositFailedEvent {
	df := DepositFailedEvent{
		FlowEvent: flowEvent,
		Reason:    reason,
	}
	for _, opt := range opts {
		opt(&df)
	}
	return &df
}

// --- DepositPersistedEvent ---

// WithDepositValidatedEventForPersisted sets the DepositValidatedEvent on a DepositPersistedEvent
func WithDepositValidatedEventForPersisted(e DepositValidatedEvent) DepositPersistedEventOpt {
	return func(d *DepositPersistedEvent) { d.DepositValidatedEvent = e }
}

// WithTransactionIDForPersisted sets the TransactionID on a DepositPersistedEvent
func WithTransactionIDForPersisted(id uuid.UUID) DepositPersistedEventOpt {
	return func(d *DepositPersistedEvent) { d.TransactionID = id }
}

// WithDepositAmountForPersisted sets the Amount on a DepositPersistedEvent
func WithDepositAmountForPersisted(amount money.Money) DepositPersistedEventOpt {
	return func(d *DepositPersistedEvent) { d.Amount = amount }
}

// NewDepositPersistedEvent creates a new DepositPersistedEvent with the given options
func NewDepositPersistedEvent(userID, accountID, correlationID uuid.UUID, opts ...DepositPersistedEventOpt) *DepositPersistedEvent {
	e := &DepositPersistedEvent{
		ID: uuid.New(),
		DepositValidatedEvent: DepositValidatedEvent{
			DepositRequestedEvent: DepositRequestedEvent{
				FlowEvent: FlowEvent{
					FlowType:      "deposit",
					UserID:        userID,
					AccountID:     accountID,
					CorrelationID: correlationID,
				},
				ID:        uuid.New(),
				Timestamp: time.Now(),
			},
		},
		TransactionID: uuid.Nil,
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// --- Usage Example (for tests) ---
// event := NewDepositRequestedEvent(userID, accountID, correlationID, WithDepositAmount(mustMoney(5000, "EUR")))
// validated := NewDepositValidatedEvent(userID, accountID, correlationID, WithDepositRequestedEvent(event))
// businessValid := NewDepositBusinessValidationEvent(userID, accountID, correlationID, WithDepositValidatedEvent(validated))
// failed := NewDepositFailedEvent(userID, accountID, correlationID, "negative amount", WithDepositBusinessValidationEvent(businessValid))
