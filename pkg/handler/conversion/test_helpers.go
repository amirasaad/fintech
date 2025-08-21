package conversion

import (
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/provider"
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
		events.WithConversionAmount(&amount),
		events.WithConversionTo(money.Code(to)),
		events.WithConversionTransactionID(transactionID),
	)

	return event
}

// NewValidConversionInfo returns a fully valid Info for use in tests.
func NewValidConversionInfo(
	originalAmount, convertedAmount float64,
	originalCurrency, convertedCurrency string,
	rate float64,
) *provider.ExchangeInfo {
	return &provider.ExchangeInfo{
		OriginalAmount:    originalAmount,
		OriginalCurrency:  originalCurrency,
		ConvertedAmount:   convertedAmount,
		ConvertedCurrency: convertedCurrency,
		ConversionRate:    rate,
	}
}
