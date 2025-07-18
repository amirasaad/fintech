package events

import (
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// DepositRequestedEvent is emitted when a deposit is requested (pure event-driven domain).
type DepositRequestedEvent struct {
	EventID   uuid.UUID
	AccountID uuid.UUID
	UserID    uuid.UUID
	Amount    money.Money
	Source    string // MoneySource as string
	Timestamp int64
}

// DepositValidatedEvent is emitted after deposit validation succeeds.
type DepositValidatedEvent struct {
	DepositRequestedEvent
	AccountID uuid.UUID
	Account   *account.Account
}

// DepositPersistedEvent is emitted after persistence is complete.
type DepositPersistedEvent struct {
	DepositValidatedEvent
	TransactionID uuid.UUID   // propagate TransactionID
	UserID        uuid.UUID   // propagate UserID
	Amount        money.Money // Amount to deposit
}

func (e DepositRequestedEvent) EventType() string { return "DepositRequestedEvent" }
func (e DepositValidatedEvent) EventType() string { return "DepositValidatedEvent" }
func (e DepositPersistedEvent) EventType() string { return "DepositPersistedEvent" }
