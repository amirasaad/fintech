package currency

import (
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/provider/exchange"
)

// CurrencyConverter is a type alias for backward compatibility.
//
// Deprecated: This type is deprecated and will be removed in a future release.
// Please use exchange.Exchange from the provider package instead.
type CurrencyConverter = exchange.Exchange

// Code represents a currency code (e.g., "USD", "EUR").
//
// Deprecated: This type is deprecated and will be removed in a future release.
// Please use money.Code from the money package instead.
type Code = money.Code

// Money represents an amount of money in a specific currency.
//
// Deprecated: This type is deprecated and will be removed in a future release.
// Please use money.Money from the money package instead.
type Money = money.Money

// Amount represents a monetary amount with its currency.
//
// Deprecated: This type is deprecated and will be removed in a future release.
// Please use money.Amount from the money package instead.
type Amount = money.Amount

// Currency represents currency information.
//
// Deprecated: This type is deprecated and will be removed in a future release.
// Please use money.Currency from the money package instead.
type Currency = money.Currency

// Common currency codes for convenience.
// These are provided for backward compatibility.
const (
	// USD represents US Dollar.
	//
	// Deprecated: Use money.USD from the money package instead.
	USD = money.USD

	// EUR represents Euro.
	//
	// Deprecated: Use money.EUR from the money package instead.
	EUR = money.EUR

	// JPY represents Japanese Yen.
	//
	// Deprecated: Use money.JPY from the money package instead.
	JPY = money.JPY

	// KWD represents Kuwaiti Dinar.
	//
	// Deprecated: Use money.KWD from the money package instead.
	KWD = money.KWD

	// GBP represents British Pound.
	//
	// Deprecated: Use money.GBP from the money package instead.
	GBP = money.GBP
)
