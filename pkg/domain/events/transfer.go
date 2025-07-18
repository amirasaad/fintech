package events

import (
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// TransferRequestedEvent is emitted when a transfer is requested (pure event-driven domain).
type TransferRequestedEvent struct {
	EventID         uuid.UUID
	SourceAccountID uuid.UUID
	DestAccountID   uuid.UUID
	SenderUserID    uuid.UUID
	ReceiverUserID  uuid.UUID
	Amount          money.Money
	Source          string // MoneySource as string
}

// TransferValidatedEvent is emitted after transfer validation succeeds.
type TransferValidatedEvent struct {
	TransferRequestedEvent
	// Add any fields produced by validation (e.g., loaded Account)
}

// TransferConversionDoneEvent is emitted after transfer currency conversion is completed.
type TransferConversionDoneEvent struct {
	ConversionDoneEvent
	SenderUserID    string
	SourceAccountID string
	TargetAccountID string
}

// TransferDomainOpDoneEvent is emitted after the transfer domain operation is complete.
type TransferDomainOpDoneEvent struct {
	TransferValidatedEvent
	SenderUserID    uuid.UUID
	SourceAccountID uuid.UUID
	Amount          money.Money
	Source          string
}

// TransferPersistedEvent is emitted after transfer persistence is complete.
type TransferPersistedEvent struct {
	TransferDomainOpDoneEvent
	// Add fields for DB transaction, etc.
}

// Legacy events for backward compatibility
type TransferConversionRequested struct {
	TransferValidatedEvent
	EventID         uuid.UUID
	TransactionID   uuid.UUID
	SourceAccountID uuid.UUID
	DestAccountID   uuid.UUID
	UserID          uuid.UUID
	Amount          money.Money
	SourceCurrency  string
	TargetCurrency  string
	Timestamp       int64
}

type TransferConversionDone struct {
	TransferConversionRequested
	ConvertedAmount money.Money
}

func (e TransferRequestedEvent) EventType() string      { return "TransferRequestedEvent" }
func (e TransferValidatedEvent) EventType() string      { return "TransferValidatedEvent" }
func (e TransferConversionDoneEvent) EventType() string { return "TransferConversionDoneEvent" }
func (e TransferDomainOpDoneEvent) EventType() string   { return "TransferDomainOpDoneEvent" }
func (e TransferPersistedEvent) EventType() string      { return "TransferPersistedEvent" }
func (e TransferConversionRequested) EventType() string { return "TransferConversionRequested" }
func (e TransferConversionDone) EventType() string      { return "TransferConversionDone" }
