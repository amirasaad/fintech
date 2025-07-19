package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/domain/money"
)

// ConversionRequestedEvent is a generic event for requesting currency conversion in any business flow.
type ConversionRequestedEvent struct {
	EventID    string
	FromAmount money.Money
	ToCurrency string
	RequestID  string
	Timestamp  time.Time
	CorrelationID string // For distributed tracing
}

// ConversionDoneEvent is a generic event for reporting the result of a currency conversion.
type ConversionDoneEvent struct {
	EventID    string
	FromAmount money.Money
	ToAmount   money.Money
	RequestID  string
	Timestamp  time.Time
	CorrelationID string // For distributed tracing
}

func (e ConversionRequestedEvent) EventType() string    { return "ConversionRequestedEvent" }
func (e ConversionDoneEvent) EventType() string         { return "ConversionDoneEvent" }
