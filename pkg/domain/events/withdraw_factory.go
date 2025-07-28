package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// --- WithdrawRequestedEvent ---
type WithdrawRequestedEventOpt func(*WithdrawRequestedEvent)

func WithWithdrawTimestamp(ts time.Time) WithdrawRequestedEventOpt {
	return func(e *WithdrawRequestedEvent) { e.Timestamp = ts }
}
func WithWithdrawID(id uuid.UUID) WithdrawRequestedEventOpt {
	return func(e *WithdrawRequestedEvent) { e.ID = id }
}
func WithWithdrawFlowEvent(fe FlowEvent) WithdrawRequestedEventOpt {
	return func(e *WithdrawRequestedEvent) { e.FlowEvent = fe }
}

func NewWithdrawRequestedEvent(userID, accountID, correlationID uuid.UUID, opts ...WithdrawRequestedEventOpt) *WithdrawRequestedEvent {
	event := WithdrawRequestedEvent{
		FlowEvent: FlowEvent{
			FlowType:      "withdraw",
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: correlationID,
		},
		ID:        uuid.New(),
		Amount:    money.Zero(currency.USD),
		Timestamp: time.Now(),
	}
	for _, opt := range opts {
		opt(&event)
	}
	return &event
}

// --- WithdrawValidatedEvent ---
type WithdrawValidatedEventOpt func(*WithdrawValidatedEvent)

func WithWithdrawRequestedEvent(e WithdrawRequestedEvent) WithdrawValidatedEventOpt {
	return func(v *WithdrawValidatedEvent) {
		// Preserve the FlowEvent fields that were already set
		e.FlowEvent = v.FlowEvent
		v.WithdrawRequestedEvent = e
	}
}
func NewWithdrawValidatedEvent(userID, accountID, correlationID uuid.UUID, opts ...WithdrawValidatedEventOpt) *WithdrawValidatedEvent {
	// Initialize WithdrawValidatedEvent with proper FlowEvent
	v := &WithdrawValidatedEvent{
		WithdrawRequestedEvent: WithdrawRequestedEvent{
			FlowEvent: FlowEvent{
				FlowType:      "withdraw",
				UserID:        userID,
				AccountID:     accountID,
				CorrelationID: correlationID,
			},
		},
	}

	// Apply any additional options
	for _, opt := range opts {
		opt(v)
	}
	return v
}
func WithTargetCurrency(s string) WithdrawValidatedEventOpt {
	return func(w *WithdrawValidatedEvent) { w.TargetCurrency = s }
}

// WithWithdrawValidatedFlowEvent sets the FlowEvent for a WithdrawValidatedEvent
func WithWithdrawValidatedFlowEvent(fe FlowEvent) WithdrawValidatedEventOpt {
	return func(v *WithdrawValidatedEvent) { v.FlowEvent = fe }
}

// --- WithdrawBusinessValidationEvent ---
type WithdrawBusinessValidationEventOpt func(*WithdrawBusinessValidationEvent)

// WithEventFlow sets the FlowEvent for a WithdrawBusinessValidationEvent
func WithEventFlow(flowType string, userID, accountID, correlationID uuid.UUID) WithdrawBusinessValidationEventOpt {
	return func(e *WithdrawBusinessValidationEvent) {
		e.FlowEvent = FlowEvent{
			FlowType:      flowType,
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: correlationID,
		}
	}
}

// WithWithdrawValidatedEvent sets the WithdrawValidatedEvent for a WithdrawBusinessValidationEvent
func WithWithdrawValidatedEvent(e WithdrawValidatedEvent) WithdrawBusinessValidationEventOpt {
	return func(b *WithdrawBusinessValidationEvent) { b.WithdrawValidatedEvent = e }
}
func NewWithdrawBusinessValidationEvent(userID, accountID, correlationID uuid.UUID, opts ...WithdrawBusinessValidationEventOpt) *WithdrawBusinessValidationEvent {
	// By default, embed a valid WithdrawValidatedEvent
	ve := NewWithdrawValidatedEvent(userID, accountID, correlationID)
	bv := WithdrawBusinessValidationEvent{
		WithdrawValidatedEvent: *ve,
		Amount:                 money.Zero(currency.USD),
	}
	for _, opt := range opts {
		opt(&bv)
	}
	return &bv
}

// --- WithdrawFailedEvent ---
type WithdrawFailedEventOpt func(*WithdrawFailedEvent)

func WithWithdrawFailureReason(reason string) WithdrawFailedEventOpt {
	return func(df *WithdrawFailedEvent) { df.Reason = reason }
}
func NewWithdrawFailedEvent(flowEvent FlowEvent, reason string, opts ...WithdrawFailedEventOpt) *WithdrawFailedEvent {
	df := WithdrawFailedEvent{
		FlowEvent: flowEvent,
		Reason:    reason,
	}
	for _, opt := range opts {
		opt(&df)
	}
	return &df
}

// --- WithdrawPersistedEvent ---
type WithdrawPersistedEventOpt func(*WithdrawPersistedEvent)

// WithWithdrawTransactionID sets the transaction ID for the WithdrawPersistedEvent
func WithWithdrawTransactionID(id uuid.UUID) WithdrawPersistedEventOpt {
	return func(e *WithdrawPersistedEvent) { e.TransactionID = id }
}

// NewWithdrawPersistedEvent creates a new WithdrawPersistedEvent with the given options
func NewWithdrawPersistedEvent(flowEvent FlowEvent, validatedEvent WithdrawValidatedEvent, opts ...WithdrawPersistedEventOpt) *WithdrawPersistedEvent {
	e := &WithdrawPersistedEvent{
		FlowEvent:              flowEvent,
		WithdrawValidatedEvent: validatedEvent,
		TransactionID:          uuid.Nil,
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}
