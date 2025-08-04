package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// CurrencyConversionRequestedOpt --- CurrencyConversionRequested ---
type CurrencyConversionRequestedOpt func(*CurrencyConversionRequested)

// WithConversionAmount sets the amount for the CurrencyConversionRequested.
func WithConversionAmount(amount money.Money) CurrencyConversionRequestedOpt {
	return func(e *CurrencyConversionRequested) { e.Amount = amount }
}

// WithConversionTo sets the target currency for the
// CurrencyConversionRequested.
func WithConversionTo(currency currency.Code) CurrencyConversionRequestedOpt {
	return func(e *CurrencyConversionRequested) { e.To = currency }
}

// WithConversionTransactionID sets the transaction ID for the
// CurrencyConversionRequested.
func WithConversionTransactionID(id uuid.UUID) CurrencyConversionRequestedOpt {
	return func(e *CurrencyConversionRequested) { e.TransactionID = id }
}

// NewCurrencyConversionRequested creates a new CurrencyConversionRequested
// with the given options.
func NewCurrencyConversionRequested(
	fe FlowEvent,
	or Event,
	opts ...CurrencyConversionRequestedOpt,
) *CurrencyConversionRequested {
	ccr := &CurrencyConversionRequested{
		FlowEvent:       fe,
		OriginalRequest: or,
	}
	ccr.ID = uuid.New()
	ccr.Timestamp = time.Now()

	for _, opt := range opts {
		opt(ccr)
	}

	return ccr
}

// CurrencyConvertedOpt --- CurrencyConverted ---
type CurrencyConvertedOpt func(*CurrencyConverted)

// NewCurrencyConverted creates a new CurrencyConverted with the given options.
func NewCurrencyConverted(
	ccr *CurrencyConversionRequested,
	opts ...CurrencyConvertedOpt,
) *CurrencyConverted {
	cc := &CurrencyConverted{
		CurrencyConversionRequested: *ccr,
	}
	cc.ID = uuid.New()
	cc.Timestamp = time.Now()

	for _, opt := range opts {
		opt(cc)
	}

	return cc
}
