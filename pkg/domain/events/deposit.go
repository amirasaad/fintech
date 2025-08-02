package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// DepositRequsted is emitted after deposit validation and persistence.
type DepositRequested struct {
	FlowEvent
	Amount        money.Money
	Source        string
	TransactionID uuid.UUID
}

func (e DepositRequested) Type() string { return "DepositRequested" }
func (e DepositRequested) Validate() error {
	return nil
}

// DepositCurrencyConverted is emitted after currency conversion for deposit.
type DepositCurrencyConverted struct {
	DepositRequested
	CurrencyConverted
	Timestamp time.Time
}

func (e DepositCurrencyConverted) Type() string { return "DepositCurrencyConverted" }

// DepositBusinessValidated is emitted after business validation for deposit.
type DepositBusinessValidated struct {
	DepositCurrencyConverted
}

func (e DepositBusinessValidated) Type() string { return "DepositBusinessValidated" }

// DepositFailed is emitted when a deposit fails.
type DepositFailed struct {
	DepositRequested
	Reason string
}

func (e DepositFailed) Type() string { return "DepositFailed" }
