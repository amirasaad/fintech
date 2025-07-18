package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// ConversionRequestedEvent is a generic event for requesting currency conversion in any business flow.
type ConversionRequestedEvent struct {
	EventID    string
	FromAmount money.Money
	ToCurrency string
	RequestID  string
	Timestamp  time.Time
}

// ConversionDoneEvent is a generic event for reporting the result of a currency conversion.
type ConversionDoneEvent struct {
	EventID    string
	FromAmount money.Money
	ToAmount   money.Money
	RequestID  string
	Timestamp  time.Time
}

// Legacy events for backward compatibility
type CurrencyConversionRequested struct {
	EventID        uuid.UUID
	TransactionID  uuid.UUID
	AccountID      uuid.UUID
	UserID         uuid.UUID
	Amount         money.Money
	SourceCurrency string
	TargetCurrency string
	Timestamp      int64
}

type CurrencyConversionDone struct {
	CurrencyConversionRequested
	ConvertedAmount money.Money
}

type CurrencyConversionPersisted struct {
	CurrencyConversionDone
}

// ConversionRequested is a generic event for requesting currency conversion in any business flow.
type ConversionRequested struct {
	CorrelationID  string      // Unique ID to correlate request/response
	FlowType       string      // "deposit", "withdraw", "transfer", etc.
	OriginalEvent  interface{} // The original event (DepositValidatedEvent, etc.)
	Amount         money.Money
	SourceCurrency string
	TargetCurrency string
	Timestamp      int64
}

// ConversionDone is a generic event for reporting the result of a currency conversion.
type ConversionDone struct {
	CorrelationID   string
	FlowType        string
	OriginalEvent   interface{}
	ConvertedAmount money.Money
	Timestamp       int64
}

func (e ConversionRequestedEvent) EventType() string    { return "ConversionRequestedEvent" }
func (e ConversionDoneEvent) EventType() string         { return "ConversionDoneEvent" }
func (e CurrencyConversionRequested) EventType() string { return "CurrencyConversionRequested" }
func (e CurrencyConversionDone) EventType() string      { return "CurrencyConversionDone" }
func (e CurrencyConversionPersisted) EventType() string { return "CurrencyConversionPersisted" }
func (e ConversionRequested) EventType() string         { return "ConversionRequested" }
func (e ConversionDone) EventType() string              { return "ConversionDone" }
