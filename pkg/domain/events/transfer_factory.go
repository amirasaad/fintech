package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// --- TransferRequested ---
type TransferRequestedOpt func(*TransferRequested)

func WithTransferRequestedAmount(m money.Money) TransferRequestedOpt {
	return func(e *TransferRequested) { e.Amount = m }
}

func WithTransferDestAccountID(id uuid.UUID) TransferRequestedOpt {
	return func(e *TransferRequested) { e.DestAccountID = id }
}

func NewTransferRequested(userID, accountID, correlationID uuid.UUID, opts ...TransferRequestedOpt) *TransferRequested {
	event := TransferRequested{
		FlowEvent: FlowEvent{
			ID:            uuid.New(),
			FlowType:      "transfer",
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: correlationID,
		},
		Amount:    money.Zero(currency.USD),
		Timestamp: time.Now(),
	}
	for _, opt := range opts {
		opt(&event)
	}
	return &event
}

type TransferCurrencyConvertedOpt func(*TransferCurrencyConverted)

// NewTransferCurrencyConverted creates a new TransferCurrencyConverted event
func NewTransferCurrencyConverted(tr *TransferRequested, opts ...TransferCurrencyConvertedOpt) *TransferCurrencyConverted {
	v := &TransferCurrencyConverted{
		TransferRequested: *tr,
	}
	v.Timestamp = time.Now()
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// --- TransferBusinessValidated ---
type TransferBusinessValidatedOpt func(*TransferBusinessValidated)

// NewTransferBusinessValidated creates a new TransferBusinessValidated event
func NewTransferBusinessValidated(tr *TransferCurrencyConverted, opts ...TransferBusinessValidatedOpt) *TransferBusinessValidated {
	v := &TransferBusinessValidated{
		TransferCurrencyConverted: *tr,
	}
	v.Timestamp = time.Now()
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// --- TransferFailed ---
type TransferFailedOpt func(*TransferFailed)

// WithReason sets the failure reason
func WithReason(reason string) TransferFailedOpt {
	return func(f *TransferFailed) { f.Reason = reason }
}

// NewTransferFailed creates a new TransferFailed event
func NewTransferFailed(flowEvent FlowEvent, reason string, opts ...TransferFailedOpt) *TransferFailed {
	f := &TransferFailed{
		TransferRequested: TransferRequested{
			FlowEvent:     flowEvent,
			TransactionID: uuid.Nil,
		},
		Reason: reason,
	}
	f.ID = uuid.New()
	f.Timestamp = time.Now()
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// --- TransferCompleted factory ---
type TransferCompletedOpt func(*TransferCompleted)

func WithTransferAmount(m money.Money) TransferCompletedOpt {
	return func(e *TransferCompleted) { e.Amount = m }
}

// NewTransferCompleted creates a new TransferCompleted with the given options
func NewTransferCompleted(tr *TransferRequested, opts ...TransferCompletedOpt) *TransferCompleted {
	event := TransferCompleted{
		TransferBusinessValidated: TransferBusinessValidated{
			TransferCurrencyConverted: TransferCurrencyConverted{
				TransferRequested: *tr,
			},
		},
	}
	for _, opt := range opts {
		opt(&event)
	}
	return &event
}
