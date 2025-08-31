package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/money"
	"github.com/google/uuid"
)

// --- WithdrawRequested ---
type WithdrawRequestedOpt func(*WithdrawRequested)

func WithWithdrawAmount(m *money.Money) WithdrawRequestedOpt {
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

// WithWithdrawBankAccountNumber sets the bank account number for the withdraw
// request
func WithWithdrawBankAccountNumber(accountNumber string) WithdrawRequestedOpt {
	return func(e *WithdrawRequested) { e.BankAccountNumber = accountNumber }
}

func NewWithdrawRequested(
	userID, accountID, correlationID uuid.UUID,
	opts ...WithdrawRequestedOpt,
) *WithdrawRequested {
	wr := &WithdrawRequested{
		FlowEvent: FlowEvent{
			ID:            uuid.New(),
			FlowType:      "withdraw",
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: correlationID,
			Timestamp:     time.Now(),
		},
		Amount: money.Zero(money.USD),
	}
	for _, opt := range opts {
		opt(wr)
	}
	return wr
}

type WithdrawCurrencyConvertedOpt func(*WithdrawCurrencyConverted)

// NewWithdrawCurrencyConverted creates a new WithdrawCurrencyConverted event.
// It takes a CurrencyConverted and combines it into a
// WithdrawCurrencyConverted event, ensuring all necessary fields are properly propagated.
func NewWithdrawCurrencyConverted(
	cc *CurrencyConverted,
	opts ...WithdrawCurrencyConvertedOpt,
) *WithdrawCurrencyConverted {
	wcc := &WithdrawCurrencyConverted{
		CurrencyConverted: *cc,
	}
	// Ensure the TransactionID is properly set from the CurrencyConverted event
	if wcc.TransactionID == uuid.Nil {
		wcc.TransactionID = cc.TransactionID
	}
	wcc.ID = uuid.New()
	wcc.Timestamp = time.Now()
	for _, opt := range opts {
		opt(wcc)
	}
	return wcc
}

type WithdrawValidatedOpt func(*WithdrawValidated)

// NewWithdrawValidated creates a new WithdrawValidated event.
// It takes a WithdrawCurrencyConverted and returns a new WithdrawValidated
// event.
func NewWithdrawValidated(
	cc *WithdrawCurrencyConverted,
	opts ...WithdrawValidatedOpt,
) *WithdrawValidated {
	wv := &WithdrawValidated{
		WithdrawCurrencyConverted: *cc,
	}
	wv.ID = uuid.New()
	wv.Timestamp = time.Now()
	for _, opt := range opts {
		opt(wv)
	}
	return wv
}

// --- WithdrawFailed ---
type WithdrawFailedOpt func(*WithdrawFailed)

func WithWithdrawFailureReason(reason string) WithdrawFailedOpt {
	return func(wf *WithdrawFailed) { wf.Reason = reason }
}

func NewWithdrawFailed(
	wr *WithdrawRequested,
	reason string,
	opts ...WithdrawFailedOpt,
) *WithdrawFailed {
	wf := &WithdrawFailed{
		WithdrawRequested: *wr,
		Reason:            reason,
	}
	wf.ID = uuid.New()
	wf.Timestamp = time.Now()
	for _, opt := range opts {
		opt(wf)
	}
	return wf
}

func NewUserOnboardingCompleted(userID uuid.UUID, stripeAccountID string) *UserOnboardingCompleted {
	return &UserOnboardingCompleted{
		FlowEvent: FlowEvent{
			ID:        uuid.New(),
			UserID:    userID,
			Timestamp: time.Now(),
		},
		StripeAccountID: stripeAccountID,
	}
}
