package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// ConversionRequestedEvent is a generic event for requesting currency conversion in any business flow.
type ConversionRequestedEvent struct {
	FlowEvent
	ID         uuid.UUID
	FromAmount money.Money
	ToCurrency string
	RequestID  string
	Timestamp  time.Time
}

// ConversionDoneEvent is a generic event for reporting the result of a currency conversion.
type ConversionDoneEvent struct {
	FlowEvent
	ID         uuid.UUID
	FromAmount money.Money
	ToAmount   money.Money
	RequestID  string
	Timestamp  time.Time
}

func (e ConversionRequestedEvent) EventType() string { return "ConversionRequestedEvent" }
func (e ConversionDoneEvent) EventType() string      { return "ConversionDoneEvent" }
