package events

import (
	"github.com/amirasaad/fintech/pkg/domain/account"
	"time"

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
	ID uuid.UUID
	DepositRequestedEvent
	Account *account.Account
}

// DepositBusinessValidationEvent is emitted after deposit currency conversion is completed.
type DepositBusinessValidationEvent struct {
	DepositValidatedEvent
	ConversionDoneEvent
	Account *account.Account
	Amount  money.Money
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
	FlowEvent
	ID uuid.UUID
	DepositBusinessValidationEvent
	TransactionID uuid.UUID
}

func (e DepositRequestedEvent) Type() string          { return "DepositRequestedEvent" }
func (e DepositValidatedEvent) Type() string          { return "DepositValidatedEvent" }
func (e DepositBusinessValidationEvent) Type() string { return "DepositBusinessValidationEvent" }
func (e DepositPersistedEvent) Type() string          { return "DepositPersistedEvent" }
func (e DepositBusinessValidatedEvent) Type() string  { return "DepositBusinessValidatedEvent" }
