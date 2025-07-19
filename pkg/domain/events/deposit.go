package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/domain/account"
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

// DepositValidatedEvent is emitted after deposit validation succeeds.
type DepositValidatedEvent struct {
	DepositRequestedEvent
	Account *account.Account
}

// DepositConversionDoneEvent is emitted after deposit currency conversion is completed.
type DepositConversionDoneEvent struct {
	DepositValidatedEvent
	ConversionDoneEvent
	TransactionID uuid.UUID
}

// DepositPersistedEvent is emitted after persistence is complete.
type DepositPersistedEvent struct {
	DepositValidatedEvent
	TransactionID uuid.UUID   // propagate TransactionID
	Amount        money.Money // Amount to deposit
}

// DepositBusinessValidatedEvent is emitted after business validation in account currency.
type DepositBusinessValidatedEvent struct {
	DepositConversionDoneEvent
	TransactionID uuid.UUID
}

func (e DepositRequestedEvent) EventType() string         { return "DepositRequestedEvent" }
func (e DepositValidatedEvent) EventType() string         { return "DepositValidatedEvent" }
func (e DepositConversionDoneEvent) EventType() string    { return "DepositConversionDoneEvent" }
func (e DepositPersistedEvent) EventType() string         { return "DepositPersistedEvent" }
func (e DepositBusinessValidatedEvent) EventType() string { return "DepositBusinessValidatedEvent" }
