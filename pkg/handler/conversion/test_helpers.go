package conversion

import (
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// NewValidConversionRequestedEvent returns a fully valid ConversionRequestedEvent for use in tests.
func NewValidConversionRequestedEvent(flow events.FlowEvent, transactionID uuid.UUID, amount money.Money, to string) *events.ConversionRequestedEvent {
	// Create the event using the factory function with options
	event := events.NewConversionRequestedEvent(
		flow,
		events.WithConversionAmount(amount),
		events.WithConversionTo(currency.Code(to)),
		events.WithConversionRequestID(uuid.New().String()),
		events.WithConversionTransactionID(transactionID),
		events.WithConversionTimestamp(time.Now()),
	)

	return event
}

// NewValidConversionInfo returns a fully valid ConversionInfo for use in tests.
func NewValidConversionInfo(originalAmount, convertedAmount float64, originalCurrency, convertedCurrency string, rate float64) *common.ConversionInfo {
	return &common.ConversionInfo{
		OriginalAmount:    originalAmount,
		OriginalCurrency:  originalCurrency,
		ConvertedAmount:   convertedAmount,
		ConvertedCurrency: convertedCurrency,
		ConversionRate:    rate,
	}
}
