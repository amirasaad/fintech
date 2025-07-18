package events

import (
	"time"

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
	Timestamp time.Time
}

// DepositValidatedEvent is emitted after deposit validation succeeds.
type DepositValidatedEvent struct {
	DepositRequestedEvent
	AccountID uuid.UUID
	Account   *account.Account
}

// DepositConversionDoneEvent is emitted after deposit currency conversion is completed.
type DepositConversionDoneEvent struct {
	ConversionDoneEvent
	UserID    string
	AccountID string
}

// DepositPersistedEvent is emitted after persistence is complete.
type DepositPersistedEvent struct {
	DepositValidatedEvent
	TransactionID uuid.UUID   // propagate TransactionID
	UserID        uuid.UUID   // propagate UserID
	Amount        money.Money // Amount to deposit
}

// Legacy events for backward compatibility
type DepositConversionRequested struct {
	DepositValidatedEvent
	EventID        uuid.UUID
	TransactionID  uuid.UUID
	AccountID      uuid.UUID
	UserID         uuid.UUID
	Amount         money.Money
	SourceCurrency string
	TargetCurrency string
	Timestamp      int64
}

type DepositConversionDone struct {
	DepositConversionRequested
	ConvertedAmount money.Money
}

func (e DepositRequestedEvent) EventType() string      { return "DepositRequestedEvent" }
func (e DepositValidatedEvent) EventType() string      { return "DepositValidatedEvent" }
func (e DepositConversionDoneEvent) EventType() string { return "DepositConversionDoneEvent" }
func (e DepositPersistedEvent) EventType() string      { return "DepositPersistedEvent" }
func (e DepositConversionRequested) EventType() string { return "DepositConversionRequested" }
func (e DepositConversionDone) EventType() string      { return "DepositConversionDone" }
