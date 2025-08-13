package events

import (
	"fmt"
	"time"

	"github.com/amirasaad/fintech/pkg/money"
	"github.com/google/uuid"
)

// WithdrawRequested is emitted when a withdrawal is requested (pure event-driven
// domain).
type WithdrawRequested struct {
	FlowEvent
	ID                    uuid.UUID
	TransactionID         uuid.UUID
	Amount                money.Money
	BankAccountNumber     string
	RoutingNumber         string
	ExternalWalletAddress string
	Timestamp             time.Time
	PaymentID             string // Added for payment provider integration
	Fee                   int64
}

func (e *WithdrawRequested) Type() string {
	return EventTypeWithdrawRequested.String()
}

// Validate performs business validation on the withdrawal request
func (e *WithdrawRequested) Validate() error {
	if e.AccountID == uuid.Nil {
		return fmt.Errorf("account ID cannot be nil")
	}
	if e.UserID == uuid.Nil {
		return fmt.Errorf("user ID cannot be nil")
	}
	if e.Amount.IsZero() {
		return fmt.Errorf("amount cannot be zero")
	}
	if e.Amount.IsNegative() {
		return fmt.Errorf("amount must be positive")
	}
	return nil
}

// WithdrawCurrencyConverted is emitted after currency conversion for withdraw.
type WithdrawCurrencyConverted struct {
	CurrencyConverted
}

func (e WithdrawCurrencyConverted) Type() string {
	return EventTypeWithdrawCurrencyConverted.String()
}

// WithdrawValidated is emitted after business validation for withdraw.
type WithdrawValidated struct {
	WithdrawCurrencyConverted
}

func (e WithdrawValidated) Type() string {
	return EventTypeWithdrawValidated.String()
}

// WithdrawFailed is emitted when any part of the withdrawal flow fails.
type WithdrawFailed struct {
	WithdrawRequested
	Reason string
}

func (e WithdrawFailed) Type() string { return EventTypeWithdrawFailed.String() }
