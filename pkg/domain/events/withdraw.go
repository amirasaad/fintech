package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/domain/account"
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
	Timestamp             time.Time
	PaymentID             string // Added for payment provider integration
}

// WithdrawValidatedEvent is emitted after withdraw validation succeeds.
type WithdrawValidatedEvent struct {
	WithdrawRequestedEvent
	TargetCurrency string
	Account        *account.Account
	// Add any fields produced by validation (e.g., loaded Account)
}

// WithdrawConversionDoneEvent is emitted after withdraw currency conversion is completed.
type WithdrawConversionDoneEvent struct {
	ConversionDoneEvent
	UserID    string
	AccountID string
	Source    string
}

// WithdrawPersistedEvent is emitted after withdraw persistence is complete.
type WithdrawPersistedEvent struct {
	WithdrawValidatedEvent
	TransactionID uuid.UUID
}
