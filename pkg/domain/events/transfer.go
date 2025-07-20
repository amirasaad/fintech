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

// TransferCompletedEvent is emitted after both tx_out and tx_in are created for an internal transfer.
type TransferCompletedEvent struct {
	TransferDomainOpDoneEvent
	TxOutID uuid.UUID
	TxInID  uuid.UUID
}

func (e TransferRequestedEvent) Type() string      { return "TransferRequestedEvent" }
func (e TransferValidatedEvent) Type() string      { return "TransferValidatedEvent" }
func (e TransferConversionDoneEvent) Type() string { return "TransferConversionDoneEvent" }
func (e TransferDomainOpDoneEvent) Type() string   { return "TransferDomainOpDoneEvent" }
func (e TransferPersistedEvent) Type() string      { return "TransferPersistedEvent" }
func (e TransferCompletedEvent) Type() string      { return "TransferCompletedEvent" }
