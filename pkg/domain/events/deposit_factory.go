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

// NewDepositRequested creates a new DepositRequested event with the given
// parameters
func NewDepositRequested(
	userID, accountID, correlationID uuid.UUID,
	opts ...DepositRequestedOpt,
) *DepositRequested {
	dr := &DepositRequested{
		FlowEvent: FlowEvent{
			ID:            uuid.New(),
			FlowType:      "deposit",
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: correlationID,
			Timestamp:     time.Now(),
		},
		TransactionID: uuid.New(),
		Amount:        money.Zero(currency.USD),
	}

	for _, opt := range opts {
		opt(dr)
	}

	return dr
}

type DepositCurrencyConvertedOpt func(*DepositCurrencyConverted)

// NewDepositCurrencyConverted creates a new DepositCurrencyConverted event with
// the given parameters
func NewDepositCurrencyConverted(
	cc *CurrencyConverted,
	opts ...DepositCurrencyConvertedOpt,
) *DepositCurrencyConverted {
	de := &DepositCurrencyConverted{
		CurrencyConverted: *cc,
	}
	de.ID = uuid.New()
	de.Timestamp = time.Now()

	for _, opt := range opts {
		opt(de)
	}

	return de
}

type DepositValidatedOpt func(*DepositValidated)

// NewDepositValidated creates a new DepositValidated event with the given parameters
func NewDepositValidated(dcv *DepositCurrencyConverted) *DepositValidated {
	dv := &DepositValidated{
		DepositCurrencyConverted: *dcv,
	}
	dv.ID = uuid.New()
	dv.Timestamp = time.Now()

	return dv
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
func NewDepositFailed(
	dr *DepositRequested,
	reason string,
	opts ...DepositFailedOpt,
) *DepositFailed {
	df := &DepositFailed{
		DepositRequested: *dr,
		Reason:           reason,
	}
	df.ID = uuid.New()
	df.Timestamp = time.Now()
	for _, opt := range opts {
		opt(df)
	}

	return df
}
