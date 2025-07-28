package events

import (
	"fmt"
	"time"

	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// TransferRequestedEvent is emitted when a transfer is requested (pure event-driven domain).
type TransferRequestedEvent struct {
	FlowEvent
	ID            uuid.UUID
	Amount        money.Money
	Source        string // MoneySource as string
	DestAccountID uuid.UUID
	Timestamp     time.Time
}

func (e TransferRequestedEvent) Type() string { return "TransferRequestedEvent" }

// TransferValidatedEvent is emitted after transfer validation succeeds.
type TransferValidatedEvent struct {
	FlowEvent
	TransferRequestedEvent
}

func (e TransferValidatedEvent) Type() string { return "TransferValidatedEvent" }

// Validate checks if the event is valid.
func (e *TransferValidatedEvent) Validate() error {
	if e.AccountID == uuid.Nil || e.UserID == uuid.Nil || e.DestAccountID == uuid.Nil || e.Amount.IsZero() || e.Amount.IsNegative() {
		return fmt.Errorf("malformed validated event: %+v", e)
	}
	return nil
}

type TransferBusinessValidationEvent struct {
	FlowEvent
	TransferValidatedEvent
	ConversionDoneEvent
	Amount money.Money
}

func (e TransferBusinessValidationEvent) Type() string { return "TransferBusinessValidationEvent" }

// TransferBusinessValidatedEvent is emitted after transfer currency conversion is completed.
type TransferBusinessValidatedEvent struct {
	FlowEvent
	TransferValidatedEvent
	ConversionDoneEvent
	Amount money.Money
}

func (e TransferBusinessValidatedEvent) Type() string { return "TransferBusinessValidatedEvent" }

// TransferDomainOpDoneEvent is emitted after the transfer domain operation is complete.
type TransferDomainOpDoneEvent struct {
	FlowEvent
	TransferValidatedEvent
	ConversionDoneEvent
	TransactionID uuid.UUID
}

func (e TransferDomainOpDoneEvent) Type() string { return "TransferDomainOpDoneEvent" }

// TransferPersistedEvent is emitted after transfer persistence is complete.
type TransferPersistedEvent struct {
	FlowEvent
	TransferDomainOpDoneEvent
}

func (e TransferPersistedEvent) Type() string { return "TransferPersistedEvent" }

// TransferCompletedEvent is emitted after both tx_out and tx_in are created for an internal transfer.
type TransferCompletedEvent struct {
	FlowEvent
	TransferDomainOpDoneEvent
	TxOutID uuid.UUID
	TxInID  uuid.UUID
}

func (e TransferCompletedEvent) Type() string { return "TransferCompletedEvent" }

// TransferFailedEvent is emitted when a transfer fails business validation or persistence.
type TransferFailedEvent struct {
	FlowEvent
	TransferRequestedEvent
	Reason string
}

func (e TransferFailedEvent) Type() string { return "TransferFailedEvent" }

// TransferEventTypes returns a map of all transfer-related event types to their constructors.
func TransferEventTypes() map[string]func() common.Event {
	return map[string]func() common.Event{
		"TransferRequestedEvent":          func() common.Event { return &TransferRequestedEvent{} },
		"TransferValidatedEvent":          func() common.Event { return &TransferValidatedEvent{} },
		"TransferBusinessValidationEvent": func() common.Event { return &TransferBusinessValidationEvent{} },
		"TransferBusinessValidatedEvent":  func() common.Event { return &TransferBusinessValidatedEvent{} },
		"TransferDomainOpDoneEvent":       func() common.Event { return &TransferDomainOpDoneEvent{} },
		"TransferPersistedEvent":          func() common.Event { return &TransferPersistedEvent{} },
		"TransferCompletedEvent":          func() common.Event { return &TransferCompletedEvent{} },
		"TransferFailedEvent":             func() common.Event { return &TransferFailedEvent{} },
	}
}
