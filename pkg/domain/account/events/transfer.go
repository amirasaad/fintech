package events

import (
	"github.com/google/uuid"
)

// TransferRequestedEvent is emitted when a transfer is requested (pure event-driven domain).
type TransferRequestedEvent struct {
	EventID         uuid.UUID
	SourceAccountID uuid.UUID
	DestAccountID   uuid.UUID
	SenderUserID    uuid.UUID
	ReceiverUserID  uuid.UUID
	Amount          float64 // main unit (e.g., dollars)
	Currency        string
	Source          string // MoneySource as string
	Timestamp       int64
}

// TransferValidatedEvent is emitted after transfer validation succeeds.
type TransferValidatedEvent struct {
	TransferRequestedEvent
	// Add any fields produced by validation (e.g., loaded Account)
}

// TransferDomainOpDoneEvent is emitted after the transfer domain operation is complete.
type TransferDomainOpDoneEvent struct {
	TransferValidatedEvent
	// Add fields for domain operation results, etc.
}

// TransferPersistedEvent is emitted after transfer persistence is complete.
type TransferPersistedEvent struct {
	TransferDomainOpDoneEvent
	// Add fields for DB transaction, etc.
}

func (e TransferRequestedEvent) EventType() string    { return "TransferRequestedEvent" }
func (e TransferValidatedEvent) EventType() string    { return "TransferValidatedEvent" }
func (e TransferDomainOpDoneEvent) EventType() string { return "TransferDomainOpDoneEvent" }
func (e TransferPersistedEvent) EventType() string    { return "TransferPersistedEvent" }
