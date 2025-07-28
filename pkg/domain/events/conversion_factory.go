package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// --- ConversionDoneEvent ---
type ConversionDoneEventOpt func(*ConversionDoneEvent)

// WithConvertedAmount sets the converted amount for the ConversionDoneEvent.
func WithConvertedAmount(amount money.Money) ConversionDoneEventOpt {
	return func(e *ConversionDoneEvent) { e.ConvertedAmount = amount }
}

// WithConversionInfo sets the conversion info for the ConversionDoneEvent.
func WithConversionInfo(info *domain.ConversionInfo) ConversionDoneEventOpt {
	return func(e *ConversionDoneEvent) { e.ConversionInfo = info }
}

// WithTransactionID sets the transaction ID for the ConversionDoneEvent.
func WithTransactionID(id uuid.UUID) ConversionDoneEventOpt {
	return func(e *ConversionDoneEvent) { e.TransactionID = id }
}

// WithRequestID sets the request ID for the ConversionDoneEvent.
func WithRequestID(id string) ConversionDoneEventOpt {
	return func(e *ConversionDoneEvent) { e.RequestID = id }
}

// WithTimestamp sets the timestamp for the ConversionDoneEvent.
func WithTimestamp(t time.Time) ConversionDoneEventOpt {
	return func(e *ConversionDoneEvent) { e.Timestamp = t }
}

// --- ConversionRequestedEvent ---
type ConversionRequestedEventOpt func(*ConversionRequestedEvent)

// WithConversionAmount sets the amount for the ConversionRequestedEvent.
func WithConversionAmount(amount money.Money) ConversionRequestedEventOpt {
	return func(e *ConversionRequestedEvent) { e.Amount = amount }
}

// WithConversionTo sets the target currency for the ConversionRequestedEvent.
func WithConversionTo(currency currency.Code) ConversionRequestedEventOpt {
	return func(e *ConversionRequestedEvent) { e.To = currency }
}

// WithConversionRequestID sets the request ID for the ConversionRequestedEvent.
func WithConversionRequestID(id string) ConversionRequestedEventOpt {
	return func(e *ConversionRequestedEvent) { e.RequestID = id }
}

// WithConversionTransactionID sets the transaction ID for the ConversionRequestedEvent.
func WithConversionTransactionID(id uuid.UUID) ConversionRequestedEventOpt {
	return func(e *ConversionRequestedEvent) { e.TransactionID = id }
}

// WithConversionTimestamp sets the timestamp for the ConversionRequestedEvent.
func WithConversionTimestamp(t time.Time) ConversionRequestedEventOpt {
	return func(e *ConversionRequestedEvent) { e.Timestamp = t }
}

// NewConversionRequestedEvent creates a new ConversionRequestedEvent with the given options.
func NewConversionRequestedEvent(
	flow FlowEvent,
	opts ...ConversionRequestedEventOpt,
) *ConversionRequestedEvent {
	event := &ConversionRequestedEvent{
		FlowEvent:     flow,
		ID:            uuid.New(),
		Timestamp:     time.Now(),
		TransactionID: uuid.Nil,
	}

	for _, opt := range opts {
		opt(event)
	}

	return event
}

// NewConversionDoneEvent creates a new ConversionDoneEvent with the given options.
func NewConversionDoneEvent(
	userID uuid.UUID,
	accountID uuid.UUID,
	correlationID uuid.UUID,
	opts ...ConversionDoneEventOpt,
) ConversionDoneEvent {
	event := ConversionDoneEvent{
		FlowEvent: FlowEvent{
			FlowType:      "conversion",
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: correlationID,
		},
		ID:        uuid.New(),
		Timestamp: time.Now(),
	}

	for _, opt := range opts {
		opt(&event)
	}

	return event
}
