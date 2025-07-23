package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"

	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// ConversionRequestedEvent is a generic event for requesting currency conversion in any business flow.
type ConversionRequestedEvent struct {
	FlowEvent
	ID            uuid.UUID
	Amount        money.Money
	To            currency.Code
	RequestID     string
	TransactionID uuid.UUID
	Timestamp     time.Time
}

// ConversionDoneEvent is a generic event for reporting the result of a currency conversion.
type ConversionDoneEvent struct {
	FlowEvent
	ID              uuid.UUID
	RequestID       string
	TransactionID   uuid.UUID
	ConvertedAmount money.Money
	ConversionInfo  *domain.ConversionInfo
	Timestamp       time.Time
}

func (e ConversionRequestedEvent) Type() string { return "ConversionRequestedEvent" }
func (e ConversionDoneEvent) Type() string      { return "ConversionDoneEvent" }
