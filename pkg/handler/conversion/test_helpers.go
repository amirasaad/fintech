package conversion

import (
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// NewValidConversionRequestedEvent returns a fully valid ConversionRequestedEvent for use in tests.
func NewValidConversionRequestedEvent(
	flow events.FlowEvent,
	transactionID uuid.UUID,
	amount money.Money,
	to string,
) *events.CurrencyConversionRequested {
	// Create the event using the factory function with options
	event := events.NewCurrencyConversionRequested(
		flow,
		nil,
		events.WithConversionAmount(amount),
		events.WithConversionTo(currency.Code(to)),
		events.WithConversionTransactionID(transactionID),
	)

	return event
}

// NewValidConversionInfo returns a fully valid ConversionInfo for use in tests.
func NewValidConversionInfo(
	originalAmount, convertedAmount float64,
	originalCurrency, convertedCurrency string,
	rate float64,
) *common.ConversionInfo {
	return &common.ConversionInfo{
		OriginalAmount:    originalAmount,
		OriginalCurrency:  originalCurrency,
		ConvertedAmount:   convertedAmount,
		ConvertedCurrency: convertedCurrency,
		ConversionRate:    rate,
	}
}
