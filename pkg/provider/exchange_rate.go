package provider

import (
	"context"
	"errors"
	"time"
)

// Common errors for exchange rate operations
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

// ExchangeInfo holds details about a currency conversion performed during a transaction.
type ExchangeInfo struct {
	OriginalAmount    float64
	OriginalCurrency  string
	ConvertedAmount   float64
	ConvertedCurrency string
	ConversionRate    float64
	Timestamp         time.Time
	Source            string
}

// ExchangeRate defines the interface for external exchange rate providers.
type ExchangeRate interface {
	// GetRate fetches the current exchange rate for a currency pair.
	GetRate(ctx context.Context, from, to string) (*ExchangeInfo, error)

	// GetRates fetches multiple exchange rates in a single request.
	GetRates(ctx context.Context, from string) (map[string]*ExchangeInfo, error)

	// IsSupported checks if a currency pair is supported by the provider.
	IsSupported(from, to string) bool

	// Name returns the provider's name for logging and identification.
	Name() string

	// IsHealthy checks if the provider is currently available.
	IsHealthy() bool
}
