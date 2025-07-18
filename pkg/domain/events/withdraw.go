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

// Legacy events for backward compatibility
type WithdrawConversionRequested struct {
	WithdrawValidatedEvent
	EventID        uuid.UUID
	TransactionID  uuid.UUID
	AccountID      uuid.UUID
	UserID         uuid.UUID
	Amount         money.Money
	SourceCurrency string
	TargetCurrency string
	Timestamp      int64
}

type WithdrawConversionDone struct {
	WithdrawConversionRequested
	ConvertedAmount money.Money
}

func (e WithdrawRequestedEvent) EventType() string      { return "WithdrawRequestedEvent" }
func (e WithdrawValidatedEvent) EventType() string      { return "WithdrawValidatedEvent" }
func (e WithdrawConversionDoneEvent) EventType() string { return "WithdrawConversionDoneEvent" }
func (e WithdrawPersistedEvent) EventType() string      { return "WithdrawPersistedEvent" }
func (e WithdrawConversionRequested) EventType() string { return "WithdrawConversionRequested" }
func (e WithdrawConversionDone) EventType() string      { return "WithdrawConversionDone" }
