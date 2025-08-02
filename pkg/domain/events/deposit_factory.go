package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// DepositRequestedOpt is a function that configures a DepositRequested
type DepositRequestedOpt func(*DepositRequested)

// WithDepositAmount sets the deposit amount
func WithDepositAmount(m money.Money) DepositRequestedOpt {
	return func(e *DepositRequested) { e.Amount = m }
}

// WithDepositTimestamp sets the deposit timestamp
func WithDepositTimestamp(ts time.Time) DepositRequestedOpt {
	return func(e *DepositRequested) { e.Timestamp = ts }
}

// WithDepositID sets the deposit ID
func WithDepositID(id uuid.UUID) DepositRequestedOpt {
	return func(e *DepositRequested) { e.ID = id }
}

// WithDepositFlowEvent sets the flow event for the deposit
func WithDepositFlowEvent(fe FlowEvent) DepositRequestedOpt {
	return func(e *DepositRequested) { e.FlowEvent = fe }
}

// WithDepositTransactionID sets the transaction ID for the deposit
func WithDepositTransactionID(id uuid.UUID) DepositRequestedOpt {
	return func(e *DepositRequested) { e.TransactionID = id }
}

// NewDepositRequested creates a new DepositRequested event with the given parameters
func NewDepositRequested(userID, accountID, correlationID uuid.UUID, opts ...DepositRequestedOpt) *DepositRequested {
	event := &DepositRequested{
		FlowEvent: FlowEvent{
			FlowType:      "deposit",
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: correlationID,
		},
		TransactionID: uuid.New(),
		Amount:        money.Zero(currency.USD),
	}
	event.ID = uuid.New()
	event.Timestamp = time.Now()

	for _, opt := range opts {
		opt(event)
	}

	return event
}

type DepositCurrencyConvertedOpt func(*DepositCurrencyConverted)

// NewDepositCurrencyConverted creates a new DepositCurrencyConverted event with the given parameters
func NewDepositCurrencyConverted(dcv *CurrencyConverted, opts ...DepositCurrencyConvertedOpt) *DepositCurrencyConverted {
	event := &DepositCurrencyConverted{
		CurrencyConverted: *dcv,
	}

	for _, opt := range opts {
		opt(event)
	}

	return event
}

type DepositBusinessValidatedOpt func(*DepositBusinessValidated)

// NewDepositBusinessValidated creates a new DepositBusinessValidated event with the given parameters
func NewDepositBusinessValidated(dcv *DepositCurrencyConverted) *DepositBusinessValidated {
	return &DepositBusinessValidated{
		DepositCurrencyConverted: *dcv,
	}
}

// DepositFailedOpt is a function that configures a DepositFailed
type DepositFailedOpt func(*DepositFailed)

// WithFailureReason sets the failure reason
func WithFailureReason(reason string) DepositFailedOpt {
	return func(df *DepositFailed) { df.Reason = reason }
}

// WithDepositFailedTransactionID sets the transaction ID for a failed deposit event
func WithDepositFailedTransactionID(id uuid.UUID) DepositFailedOpt {
	return func(df *DepositFailed) { df.TransactionID = id }
}

// NewDepositFailed creates a new DepositFailed event with the given parameters
func NewDepositFailed(requsted DepositRequested, reason string, opts ...DepositFailedOpt) *DepositFailed {
	failed := &DepositFailed{
		DepositRequested: requsted,
		Reason:           reason,
	}
	failed.ID = uuid.New()
	failed.Timestamp = time.Now()
	for _, opt := range opts {
		opt(failed)
	}

	return failed
}
