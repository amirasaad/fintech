package events

import (
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// TransferRequestedEvent is emitted when a transfer is requested (pure event-driven domain).
type TransferRequestedEvent struct {
	FlowEvent
	ID             uuid.UUID
	Amount         money.Money
	Source         string // MoneySource as string
	DestAccountID  uuid.UUID
	ReceiverUserID uuid.UUID
}

// TransferValidatedEvent is emitted after transfer validation succeeds.
type TransferValidatedEvent struct {
	TransferRequestedEvent
}

// TransferConversionDoneEvent is emitted after transfer currency conversion is completed.
type TransferConversionDoneEvent struct {
	TransferValidatedEvent
	ConversionDoneEvent
}

// TransferDomainOpDoneEvent is emitted after the transfer domain operation is complete.
type TransferDomainOpDoneEvent struct {
	TransferValidatedEvent
}

// TransferPersistedEvent is emitted after transfer persistence is complete.
type TransferPersistedEvent struct {
	TransferDomainOpDoneEvent
}

func (e TransferRequestedEvent) EventType() string      { return "TransferRequestedEvent" }
func (e TransferValidatedEvent) EventType() string      { return "TransferValidatedEvent" }
func (e TransferConversionDoneEvent) EventType() string { return "TransferConversionDoneEvent" }
func (e TransferDomainOpDoneEvent) EventType() string   { return "TransferDomainOpDoneEvent" }
func (e TransferPersistedEvent) EventType() string      { return "TransferPersistedEvent" }
