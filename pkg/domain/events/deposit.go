package events

import (
	"errors"
	"time"

	"github.com/amirasaad/fintech/pkg/domain/common"

	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// DepositRequestedEvent is emitted when a deposit is requested (pure event-driven domain).
type DepositRequestedEvent struct {
	FlowEvent
	ID        uuid.UUID
	Amount    money.Money
	Source    string // MoneySource as string
	Timestamp time.Time
}

func (e DepositRequestedEvent) Validate() error {
	if e.Amount.IsNegative() {
		return errors.New("amount must be positive")
	}
	return nil
}

// DepositValidatedEvent is emitted after deposit validation succeeds.
type DepositValidatedEvent struct {
	DepositRequestedEvent
	ID            uuid.UUID
	TransactionID uuid.UUID
}

// DepositBusinessValidationEvent is emitted after deposit currency conversion is completed.
type DepositBusinessValidationEvent struct {
	DepositValidatedEvent
	ConversionDoneEvent
	Amount money.Money
}

// DepositPersistedEvent is emitted after persistence is complete.
type DepositPersistedEvent struct {
	ID uuid.UUID
	DepositValidatedEvent
	TransactionID uuid.UUID   // propagate TransactionID
	Amount        money.Money // Amount to deposit
}

// DepositBusinessValidatedEvent is emitted after business validation in account currency.
type DepositBusinessValidatedEvent struct {
	DepositBusinessValidationEvent
	ID            uuid.UUID
	TransactionID uuid.UUID
}

type DepositFailedEvent struct {
	FlowEvent
	TransactionID uuid.UUID
	Reason        string
}

func (e DepositRequestedEvent) Type() string          { return "DepositRequestedEvent" }
func (e DepositValidatedEvent) Type() string          { return "DepositValidatedEvent" }
func (e DepositBusinessValidationEvent) Type() string { return "DepositBusinessValidationEvent" }
func (e DepositPersistedEvent) Type() string          { return "DepositPersistedEvent" }
func (e DepositBusinessValidatedEvent) Type() string  { return "DepositBusinessValidatedEvent" }
func (e DepositFailedEvent) Type() string             { return "DepositFailedEvent" }

func DepositEventTypes() map[string]func() common.Event {
	return map[string]func() common.Event{
		"DepositRequestedEvent":          func() common.Event { return &DepositRequestedEvent{} },
		"DepositValidatedEvent":          func() common.Event { return &DepositValidatedEvent{} },
		"DepositBusinessValidationEvent": func() common.Event { return &DepositBusinessValidationEvent{} },
		"DepositPersistedEvent":          func() common.Event { return &DepositPersistedEvent{} },
		"DepositBusinessValidatedEvent":  func() common.Event { return &DepositBusinessValidatedEvent{} },
		"DepositFailedEvent":             func() common.Event { return &DepositFailedEvent{} },
	}
}
