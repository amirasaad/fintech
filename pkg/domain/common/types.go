package common

import (
	"errors"

	"github.com/amirasaad/fintech/pkg/money"
)

// ErrInvalidCurrencyCode is returned when a currency code is invalid.
// Deprecated: Use money.ErrInvalidCurrency instead
var ErrInvalidCurrencyCode = money.ErrInvalidCurrency

// ErrUnsupportedCurrency is return when currency
// is not supported by global registry #see: /pkg/currency
var ErrUnsupportedCurrency = errors.New("currency not supported")

// ErrInvalidDecimalPlaces is returned when a monetary amount
// has more decimal places than allowed by the currency.
// Deprecated: use currency.ErrInvalidDecimals
var ErrInvalidDecimalPlaces = money.ErrInvalidAmount
