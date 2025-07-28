package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// --- TransferRequestedEvent ---
type TransferRequestedEventOpt func(*TransferRequestedEvent)

func WithTransferRequestedAmount(m money.Money) TransferRequestedEventOpt {
	return func(e *TransferRequestedEvent) { e.Amount = m }
}

func WithTransferDestAccountID(id uuid.UUID) TransferRequestedEventOpt {
	return func(e *TransferRequestedEvent) { e.DestAccountID = id }
}

func NewTransferRequestedEvent(userID, accountID, correlationID uuid.UUID, opts ...TransferRequestedEventOpt) *TransferRequestedEvent {
	event := TransferRequestedEvent{
		FlowEvent: FlowEvent{
			FlowType:      "transfer",
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

// --- TransferValidatedEvent ---
type TransferValidatedEventOpt func(*TransferValidatedEvent)

func WithTransferRequestedEvent(e TransferRequestedEvent) TransferValidatedEventOpt {
	return func(v *TransferValidatedEvent) { v.TransferRequestedEvent = e }
}

func NewTransferValidatedEvent(userID, accountID, correlationID uuid.UUID, opts ...TransferValidatedEventOpt) *TransferValidatedEvent {
	// By default, embed a valid TransferRequestedEvent
	tre := NewTransferRequestedEvent(userID, accountID, correlationID)
	v := TransferValidatedEvent{
		TransferRequestedEvent: *tre,
	}
	for _, opt := range opts {
		opt(&v)
	}
	return &v
}

// --- TransferBusinessValidationEvent ---
type TransferBusinessValidationEventOpt func(*TransferBusinessValidationEvent)

func WithTransferValidatedEvent(e TransferValidatedEvent) TransferBusinessValidationEventOpt {
	return func(bv *TransferBusinessValidationEvent) { bv.TransferValidatedEvent = e }
}

func WithTransferBusinessValidationAmount(m money.Money) TransferBusinessValidationEventOpt {
	return func(bv *TransferBusinessValidationEvent) { bv.Amount = m }
}

func NewTransferBusinessValidationEvent(userID, accountID, correlationID uuid.UUID, opts ...TransferBusinessValidationEventOpt) *TransferBusinessValidationEvent {
	event := TransferBusinessValidationEvent{
		TransferValidatedEvent: *NewTransferValidatedEvent(userID, accountID, correlationID),
		Amount:                 money.Zero(currency.USD),
	}
	for _, opt := range opts {
		opt(&event)
	}
	return &event
}

// --- TransferFailedEvent ---
type TransferFailedEventOpt func(*TransferFailedEvent)

func WithTransferFailedRequestedEvent(e TransferRequestedEvent) TransferFailedEventOpt {
	return func(tf *TransferFailedEvent) { tf.TransferRequestedEvent = e }
}
func WithTransferFailedReason(reason string) TransferFailedEventOpt {
	return func(tf *TransferFailedEvent) { tf.Reason = reason }
}

func NewTransferFailedEvent(userID, accountID, correlationID uuid.UUID, reason string, opts ...TransferFailedEventOpt) *TransferFailedEvent {
	event := TransferFailedEvent{
		TransferRequestedEvent: *NewTransferRequestedEvent(userID, accountID, correlationID),
		Reason:                 reason,
	}
	for _, opt := range opts {
		opt(&event)
	}
	return &event
}

// --- TransferDomainOpDoneEvent factory ---
type TransferDomainOpDoneEventOpt func(*TransferDomainOpDoneEvent)

func WithTransferAmount(m money.Money) TransferDomainOpDoneEventOpt {
	return func(e *TransferDomainOpDoneEvent) { e.Amount = m }
}

func WithTransferTimestamp(ts time.Time) TransferDomainOpDoneEventOpt {
	return func(e *TransferDomainOpDoneEvent) { e.Timestamp = ts }
}

func WithTransferID(id uuid.UUID) TransferDomainOpDoneEventOpt {
	return func(e *TransferDomainOpDoneEvent) { e.ID = id }
}

func WithDestAccountID(id uuid.UUID) TransferDomainOpDoneEventOpt {
	return func(e *TransferDomainOpDoneEvent) { e.DestAccountID = id }
}

func WithTransferFlowEvent(fe FlowEvent) TransferDomainOpDoneEventOpt {
	return func(e *TransferDomainOpDoneEvent) { e.FlowEvent = fe }
}

// NewTransferDomainOpDoneEvent creates a new TransferDomainOpDoneEvent with the given options
func NewTransferDomainOpDoneEvent(opts ...TransferDomainOpDoneEventOpt) *TransferDomainOpDoneEvent {
	event := &TransferDomainOpDoneEvent{}
	for _, opt := range opts {
		opt(event)
	}
	return event
}

// --- TransferCompletedEvent factory ---
type TransferCompletedEventOpt func(*TransferCompletedEvent)

// WithTransferDomainOpDoneEvent sets the base TransferDomainOpDoneEvent for the TransferCompletedEvent
func WithTransferDomainOpDoneEvent(e TransferDomainOpDoneEvent) TransferCompletedEventOpt {
	return func(t *TransferCompletedEvent) { t.TransferDomainOpDoneEvent = e }
}

// WithTxOutID sets the TxOutID for the TransferCompletedEvent
func WithTxOutID(id uuid.UUID) TransferCompletedEventOpt {
	return func(e *TransferCompletedEvent) { e.TxOutID = id }
}

// WithTxInID sets the TxInID for the TransferCompletedEvent
func WithTxInID(id uuid.UUID) TransferCompletedEventOpt {
	return func(e *TransferCompletedEvent) { e.TxInID = id }
}

// NewTransferCompletedEvent creates a new TransferCompletedEvent with the given options
func NewTransferCompletedEvent(opts ...TransferCompletedEventOpt) *TransferCompletedEvent {
	event := &TransferCompletedEvent{
		TxOutID: uuid.Nil,
		TxInID:  uuid.Nil,
	}
	for _, opt := range opts {
		opt(event)
	}
	return event
}
