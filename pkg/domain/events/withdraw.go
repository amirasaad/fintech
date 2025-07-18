package events

import (
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// WithdrawRequestedEvent is emitted when a withdrawal is requested (pure event-driven domain).
type WithdrawRequestedEvent struct {
	EventID               uuid.UUID
	AccountID             uuid.UUID
	UserID                uuid.UUID
	Amount                money.Money
	BankAccountNumber     string
	RoutingNumber         string
	ExternalWalletAddress string
	Timestamp             int64
	PaymentID             string // Added for payment provider integration
}

// WithdrawValidatedEvent is emitted after withdraw validation succeeds.
type WithdrawValidatedEvent struct {
	WithdrawRequestedEvent
	// Add any fields produced by validation (e.g., loaded Account)
}

// WithdrawPersistedEvent is emitted after withdraw persistence is complete.
type WithdrawPersistedEvent struct {
	WithdrawValidatedEvent
	TransactionID uuid.UUID
}

func (e WithdrawRequestedEvent) EventType() string { return "WithdrawRequestedEvent" }
func (e WithdrawValidatedEvent) EventType() string { return "WithdrawValidatedEvent" }
func (e WithdrawPersistedEvent) EventType() string { return "WithdrawPersistedEvent" }
