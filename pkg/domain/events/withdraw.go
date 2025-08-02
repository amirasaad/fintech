package events

import (
	"fmt"
	"time"

	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// WithdrawRequestedEvent is emitted when a withdrawal is requested (pure event-driven domain).
type WithdrawRequested struct {
	FlowEvent
	ID                    uuid.UUID
	Amount                money.Money
	BankAccountNumber     string
	RoutingNumber         string
	ExternalWalletAddress string
	Timestamp             time.Time
	PaymentID             string // Added for payment provider integration
}

func (e WithdrawRequested) Type() string { return "WithdrawRequested" }

// Validate performs business validation on the withdraw request
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
	WithdrawRequested
	CurrencyConverted
}

func (e WithdrawCurrencyConverted) Type() string { return "WithdrawCurrencyConverted" }

// WithdrawBusinessValidated is emitted after business validation for withdraw.
type WithdrawBusinessValidated struct {
	WithdrawCurrencyConverted
}

func (e WithdrawBusinessValidated) Type() string { return "WithdrawBusinessValidated" }

// WithdrawFailed is emitted when any part of the withdrawal flow fails.
type WithdrawFailed struct {
	WithdrawRequested
	Reason string
}

func (e WithdrawFailed) Type() string { return "WithdrawFailed" }
