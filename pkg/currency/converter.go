package currency

import (
	"errors"
	"time"
)

var (
	// ErrExchangeRateUnavailable indicates the exchange rate service is unreachable or down.
	ErrExchangeRateUnavailable = errors.New("exchange rate service unavailable")

	// ErrUnsupportedCurrencyPair indicates the currency pair is not supported.
	ErrUnsupportedCurrencyPair = errors.New("unsupported currency pair")

	// ErrExchangeRateExpired indicates the exchange rate data is stale or expired.
	ErrExchangeRateExpired = errors.New("exchange rate has expired")

	// ErrExchangeRateInvalid indicates the received exchange rate is invalid.
	ErrExchangeRateInvalid = errors.New("invalid exchange rate received")
)

// Converter defines the interface for converting amounts between currencies.
type Converter interface {
	// Convert converts an amount from one currency to another.
	// Returns the converted amount and the rate used, or an error if conversion is not possible.
	Convert(amount float64, from Code, to Code) (*Info, error)

	// GetRate returns the current exchange rate between two currencies.
	// This is useful for displaying rates without performing a conversion.
	GetRate(from, to string) (float64, error)

	// IsSupported checks if a currency pair is supported by the converter.
	IsSupported(from, to string) bool
}

// ExchangeRate represents a single exchange rate with metadata.
type ExchangeRate struct {
	FromCurrency string
	ToCurrency   string
	Rate         float64
	LastUpdated  time.Time
	Source       string
	ExpiresAt    time.Time
}

// Info holds details about a currency conversion performed during a transaction.
type Info struct {
	OriginalAmount    float64
	OriginalCurrency  string
	ConvertedAmount   float64
	ConvertedCurrency string
	ConversionRate    float64
}
