package events

import (
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// DepositRequested is emitted after deposit validation and persistence.
type DepositRequested struct {
	FlowEvent
	Amount        money.Money
	Source        string
	TransactionID uuid.UUID
}

func (e DepositRequested) Type() string { return EventTypeDepositRequested.String() }
func (e DepositRequested) Validate() error {
	return nil
}

// DepositCurrencyConverted is emitted after currency conversion for deposit.
type DepositCurrencyConverted struct {
	CurrencyConverted
}

func (e DepositCurrencyConverted) Type() string {
	return EventTypeDepositCurrencyConverted.String()
}

// DepositValidated is emitted after business validation for deposit.
type DepositValidated struct {
	DepositCurrencyConverted
}

func (e DepositValidated) Type() string { return EventTypeDepositValidated.String() }

// DepositFailed is emitted when a deposit fails.
type DepositFailed struct {
	DepositRequested
	Reason string
}

func (e DepositFailed) Type() string { return EventTypeDepositFailed.String() }
