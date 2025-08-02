package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// --- WithdrawRequested ---
type WithdrawRequestedOpt func(*WithdrawRequested)

func WithWithdrawAmount(m money.Money) WithdrawRequestedOpt {
	return func(e *WithdrawRequested) { e.Amount = m }
}

func WithWithdrawTimestamp(ts time.Time) WithdrawRequestedOpt {
	return func(e *WithdrawRequested) { e.Timestamp = ts }
}

func WithWithdrawID(id uuid.UUID) WithdrawRequestedOpt {
	return func(e *WithdrawRequested) { e.ID = id }
}

func WithWithdrawFlowEvent(fe FlowEvent) WithdrawRequestedOpt {
	return func(e *WithdrawRequested) { e.FlowEvent = fe }
}

// WithWithdrawBankAccountNumber sets the bank account number for the withdraw request
func WithWithdrawBankAccountNumber(accountNumber string) WithdrawRequestedOpt {
	return func(e *WithdrawRequested) { e.BankAccountNumber = accountNumber }
}

func NewWithdrawRequested(userID, accountID, correlationID uuid.UUID, opts ...WithdrawRequestedOpt) *WithdrawRequested {
	event := WithdrawRequested{
		FlowEvent: FlowEvent{
			FlowType:      "withdraw",
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

type WithdrawCurrencyConvertedOpt func(*WithdrawCurrencyConverted)

// NewWithdrawCurrencyConverted creates a new WithdrawCurrencyConverted event.
// It takes a WithdrawRequested and a CurrencyConverted and combines them into a WithdrawCurrencyConverted event.
func NewWithdrawCurrencyConverted(withdrawReq *WithdrawRequested, converted *CurrencyConverted, opts ...WithdrawCurrencyConvertedOpt) *WithdrawCurrencyConverted {
	wr := WithdrawCurrencyConverted{
		WithdrawRequested: *withdrawReq,
		CurrencyConverted: *converted,
	}
	wr.Timestamp = time.Now()
	for _, opt := range opts {
		opt(&wr)
	}
	return &wr
}

type WithdrawBusinessValidatedOpt func(*WithdrawBusinessValidated)

// NewWithdrawBusinessValidated creates a new WithdrawBusinessValidated event.
// It takes a WithdrawCurrencyConverted and returns a new WithdrawBusinessValidated event.
func NewWithdrawBusinessValidated(converted *WithdrawCurrencyConverted, opts ...WithdrawBusinessValidatedOpt) *WithdrawBusinessValidated {
	wbt := WithdrawBusinessValidated{
		WithdrawCurrencyConverted: *converted,
	}
	wbt.Timestamp = time.Now()
	for _, opt := range opts {
		opt(&wbt)
	}
	return &wbt
}

// --- WithdrawFailed ---
type WithdrawFailedOpt func(*WithdrawFailed)

func WithWithdrawFailureReason(reason string) WithdrawFailedOpt {
	return func(df *WithdrawFailed) { df.Reason = reason }
}

func NewWithdrawFailed(requested *WithdrawRequested, reason string, opts ...WithdrawFailedOpt) *WithdrawFailed {
	df := WithdrawFailed{
		WithdrawRequested: *requested,
		Reason:            reason,
	}
	for _, opt := range opts {
		opt(&df)
	}
	return &df
}
