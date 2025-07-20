package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// ConversionRequestedEvent is a generic event for requesting currency conversion in any business flow.
type ConversionRequestedEvent struct {
	FlowEvent
	ID               uuid.UUID
	FromAmount       money.Money
	ToCurrency       string
	RequestID        string
	TransactionID    uuid.UUID
	Timestamp        time.Time
	ConversionRate   float64
	OriginalCurrency string
	ConvertedAmount  float64
}

// ConversionDoneEvent is a generic event for reporting the result of a currency conversion.
type ConversionDoneEvent struct {
	FlowEvent
	ID               uuid.UUID
	FromAmount       money.Money
	ToAmount         money.Money
	RequestID        string
	TransactionID    uuid.UUID
	Timestamp        time.Time
	ConversionRate   float64
	OriginalCurrency string
	ConvertedAmount  float64
}

func (e ConversionRequestedEvent) Type() string { return "ConversionRequestedEvent" }
func (e ConversionDoneEvent) Type() string      { return "ConversionDoneEvent" }
