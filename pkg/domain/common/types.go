package common

import (
	"github.com/amirasaad/fintech/pkg/currency"
)

type ConversionInfo currency.Info

// ErrInvalidCurrencyCode is returned when a currency code is invalid.
var ErrInvalidCurrencyCode = currency.ErrInvalidCode

// ErrUnsupportedCurrency is return when currency
// is not supported by global registry #see: /pkg/currency
var ErrUnsupportedCurrency = currency.ErrUnsupported

// ErrInvalidDecimalPlaces is returned when a monetary amount
// has more decimal places than allowed by the currency.
var ErrInvalidDecimalPlaces = currency.ErrInvalidDecimals
