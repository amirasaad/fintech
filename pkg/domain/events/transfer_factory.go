package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/money"
	"github.com/google/uuid"
)

// --- TransferRequested ---
type TransferRequestedOpt func(*TransferRequested)

func WithTransferRequestedAmount(m *money.Money) TransferRequestedOpt {
	return func(e *TransferRequested) { e.Amount = m }
}

// WithTransferFee sets the transfer fee
func WithTransferFee(fee int64) TransferRequestedOpt {
	return func(e *TransferRequested) { e.Fee = fee }
}

func WithTransferDestAccountID(id uuid.UUID) TransferRequestedOpt {
	return func(e *TransferRequested) { e.DestAccountID = id }
}

func NewTransferRequested(
	userID, accountID, correlationID uuid.UUID,
	opts ...TransferRequestedOpt,
) *TransferRequested {
	event := TransferRequested{
		FlowEvent: FlowEvent{
			ID:            uuid.New(),
			FlowType:      "transfer",
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: correlationID,
		},
		Amount:    money.Zero(money.USD),
		Timestamp: time.Now(),
	}
	for _, opt := range opts {
		opt(&event)
	}
	return &event
}

type TransferCurrencyConvertedOpt func(*TransferCurrencyConverted)

// NewTransferCurrencyConverted creates a new TransferCurrencyConverted event
func NewTransferCurrencyConverted(
	cc *CurrencyConverted,
	opts ...TransferCurrencyConvertedOpt,
) *TransferCurrencyConverted {
	tr := &TransferCurrencyConverted{
		CurrencyConverted: *cc,
	}
	for _, opt := range opts {
		opt(tr)
	}
	return tr
}

// TransferBusinessValidatedOpt --- TransferBusinessValidated ---
type TransferBusinessValidatedOpt func(*TransferValidated)

// NewTransferBusinessValidated creates a new TransferBusinessValidated event
func NewTransferBusinessValidated(
	tr *TransferCurrencyConverted,
	opts ...TransferBusinessValidatedOpt,
) *TransferValidated {
	tf := &TransferValidated{
		TransferCurrencyConverted: *tr,
	}
	tf.Timestamp = time.Now()
	for _, opt := range opts {
		opt(tf)
	}
	return tf
}

// TransferFailedOpt --- TransferFailed ---
type TransferFailedOpt func(*TransferFailed)

// WithReason sets the failure reason
func WithReason(reason string) TransferFailedOpt {
	return func(f *TransferFailed) { f.Reason = reason }
}

// NewTransferFailed creates a new TransferFailed event
func NewTransferFailed(
	tr *TransferRequested,
	reason string,
	opts ...TransferFailedOpt,
) *TransferFailed {
	tf := &TransferFailed{
		TransferRequested: *tr,
		Reason:            reason,
	}
	tf.ID = uuid.New()
	tf.Timestamp = time.Now()
	for _, opt := range opts {
		opt(tf)
	}
	return tf
}

// TransferCompletedOpt --- TransferCompleted factory ---
type TransferCompletedOpt func(*TransferCompleted)

func WithTransferAmount(m *money.Money) TransferCompletedOpt {
	return func(e *TransferCompleted) { e.Amount = m }
}

// NewTransferCompleted creates a new TransferCompleted with the given options
func NewTransferCompleted(
	tr *TransferRequested,
	opts ...TransferCompletedOpt,
) *TransferCompleted {
	tc := &TransferCompleted{
		TransferValidated: TransferValidated{
			TransferCurrencyConverted: TransferCurrencyConverted{
				CurrencyConverted: *NewCurrencyConverted(
					NewCurrencyConversionRequested(tr.FlowEvent, tr),
				),
			},
		},
	}
	tc.ID = uuid.New()
	tc.Timestamp = time.Now()
	for _, opt := range opts {
		opt(tc)
	}
	return tc
}
