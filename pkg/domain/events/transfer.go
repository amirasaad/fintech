package events

import (
	"fmt"
	"time"

	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// TransferRequested is emitted after transfer validation and persistence.
type TransferRequested struct {
	FlowEvent
	Amount        money.Money
	Source        string
	DestAccountID uuid.UUID
	Timestamp     time.Time
	TransactionID uuid.UUID
}

func (e TransferRequested) Type() string { return "TransferRequested" }

// Validate checks if the event is valid.
func (e *TransferRequested) Validate() error {
	if e.AccountID == uuid.Nil || e.UserID == uuid.Nil || e.DestAccountID == uuid.Nil || e.Amount.IsZero() || e.Amount.IsNegative() {
		return fmt.Errorf("malformed validated event: %+v", e)
	}
	return nil
}

// TransferCurrencyConverted is emitted after currency conversion for transfer.
type TransferCurrencyConverted struct {
	TransferRequested
	CurrencyConverted
}

func (e TransferCurrencyConverted) Type() string { return "TransferCurrencyConverted" }

// TransferBusinessValidated is emitted after business validation for transfer.
type TransferBusinessValidated struct {
	TransferCurrencyConverted
}

func (e TransferBusinessValidated) Type() string { return "TransferBusinessValidated" }

// TransferCompleted is emitted when transfer is fully completed.
type TransferCompleted struct {
	TransferBusinessValidated
}

func (e TransferCompleted) Type() string { return "TransferCompleted" }

// TransferFailed is emitted when transfer fails.
type TransferFailed struct {
	TransferRequested
	Reason string
}

func (e TransferFailed) Type() string { return "TransferFailed" }
