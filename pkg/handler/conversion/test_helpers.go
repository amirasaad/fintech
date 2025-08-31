package conversion

import (
	"time"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/provider/exchange"
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
	fromCurrency, toCurrency string,
	rateValue float64,
) *exchange.RateInfo {
	return &exchange.RateInfo{
		FromCurrency: fromCurrency,
		ToCurrency:   toCurrency,
		Rate:         rateValue,
		Timestamp:    time.Now(), // Add timestamp
		Provider:     "test",     // Add provider
	}
}
