package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// WithdrawRequestedEvent is emitted when a withdrawal is requested (pure event-driven domain).
type WithdrawRequestedEvent struct {
	FlowEvent
	ID                    uuid.UUID
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
}

// WithdrawConversionDoneEvent is emitted after withdraw currency conversion is completed.
type WithdrawConversionDoneEvent struct {
	WithdrawValidatedEvent
	ConversionDoneEvent
}

// WithdrawPersistedEvent is emitted after withdraw persistence is complete.
type WithdrawPersistedEvent struct {
	WithdrawValidatedEvent
	TransactionID uuid.UUID
}

// WithdrawBusinessValidatedEvent is emitted after business validation in account currency for withdraw.
type WithdrawBusinessValidatedEvent struct {
	WithdrawConversionDoneEvent
	TransactionID uuid.UUID
}

func (e WithdrawRequestedEvent) Type() string      { return "WithdrawRequestedEvent" }
func (e WithdrawValidatedEvent) Type() string      { return "WithdrawValidatedEvent" }
func (e WithdrawConversionDoneEvent) Type() string { return "WithdrawConversionDoneEvent" }
func (e WithdrawPersistedEvent) Type() string      { return "WithdrawPersistedEvent" }
func (e WithdrawBusinessValidatedEvent) Type() string { return "WithdrawBusinessValidatedEvent" }
