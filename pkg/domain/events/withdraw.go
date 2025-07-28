package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/domain/common"
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

func (e WithdrawRequestedEvent) Type() string        { return "WithdrawRequestedEvent" }
func (e WithdrawRequestedEvent) FlowData() FlowEvent { return e.FlowEvent }

// WithdrawValidatedEvent is emitted after withdraw validation succeeds.
type WithdrawValidatedEvent struct {
	FlowEvent
	WithdrawRequestedEvent
	TargetCurrency string
}

func (e WithdrawValidatedEvent) Type() string        { return "WithdrawValidatedEvent" }
func (e WithdrawValidatedEvent) FlowData() FlowEvent { return e.FlowEvent }

// WithdrawBusinessValidationEvent is emitted after withdraw currency conversion is completed.
type WithdrawBusinessValidationEvent struct {
	FlowEvent
	WithdrawValidatedEvent
	ConversionDoneEvent
	Amount money.Money
}

func (e WithdrawBusinessValidationEvent) Type() string        { return "WithdrawBusinessValidationEvent" }
func (e WithdrawBusinessValidationEvent) FlowData() FlowEvent { return e.FlowEvent }

// WithdrawPersistedEvent is emitted after withdraw persistence is complete.
type WithdrawPersistedEvent struct {
	FlowEvent
	WithdrawValidatedEvent
	TransactionID uuid.UUID
}

func (e WithdrawPersistedEvent) Type() string        { return "WithdrawPersistedEvent" }
func (e WithdrawPersistedEvent) FlowData() FlowEvent { return e.FlowEvent }

// WithdrawBusinessValidatedEvent is emitted after business validation in account currency for withdraw.
type WithdrawBusinessValidatedEvent struct {
	FlowEvent
	WithdrawBusinessValidationEvent
	TransactionID uuid.UUID
}

func (e WithdrawBusinessValidatedEvent) Type() string        { return "WithdrawBusinessValidatedEvent" }
func (e WithdrawBusinessValidatedEvent) FlowData() FlowEvent { return e.FlowEvent }

// WithdrawFailedEvent is emitted when any part of the withdrawal flow fails.
type WithdrawFailedEvent struct {
	FlowEvent
	WithdrawRequestedEvent
	Reason string
}

func (e WithdrawFailedEvent) Type() string        { return "WithdrawFailedEvent" }
func (e WithdrawFailedEvent) FlowData() FlowEvent { return e.FlowEvent }

// WithdrawEventTypes returns a map of all withdraw-related event types to their constructors.
func WithdrawEventTypes() map[string]func() common.Event {
	return map[string]func() common.Event{
		"WithdrawRequestedEvent":          func() common.Event { return &WithdrawRequestedEvent{} },
		"WithdrawValidatedEvent":          func() common.Event { return &WithdrawValidatedEvent{} },
		"WithdrawBusinessValidationEvent": func() common.Event { return &WithdrawBusinessValidationEvent{} },
		"WithdrawPersistedEvent":          func() common.Event { return &WithdrawPersistedEvent{} },
		"WithdrawBusinessValidatedEvent":  func() common.Event { return &WithdrawBusinessValidatedEvent{} },
		"WithdrawFailedEvent":             func() common.Event { return &WithdrawFailedEvent{} },
	}
}
