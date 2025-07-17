package events

import (
	"github.com/google/uuid"
)

// DepositRequestedEvent is emitted when a deposit is requested (pure event-driven domain).
type DepositRequestedEvent struct {
	EventID   uuid.UUID
	AccountID string
	UserID    string
	Amount    int64
	Currency  string
	Source    string // MoneySource as string
	Timestamp int64
	PaymentID string // Added for payment provider integration
}

// DepositValidatedEvent is emitted after deposit validation succeeds.
type DepositValidatedEvent struct {
	DepositRequestedEvent
	AccountID string
}

// MoneyCreatedEvent is emitted after money creation/conversion.
type MoneyCreatedEvent struct {
	DepositValidatedEvent
	Amount         int64 // Amount in smallest unit (e.g., cents)
	Currency       string
	TargetCurrency string
}

// DepositPersistedEvent is emitted after persistence is complete.
type DepositPersistedEvent struct {
	MoneyCreatedEvent
	// Add fields for DB transaction, etc.
}

func (e DepositRequestedEvent) EventType() string { return "DepositRequestedEvent" }
func (e DepositValidatedEvent) EventType() string { return "DepositValidatedEvent" }
func (e MoneyCreatedEvent) EventType() string     { return "MoneyCreatedEvent" }
func (e DepositPersistedEvent) EventType() string { return "DepositPersistedEvent" }
