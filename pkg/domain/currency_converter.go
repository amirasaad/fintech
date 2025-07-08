package domain

import (
	"errors"
	"time"
)

var (
	ErrExchangeRateUnavailable = errors.New("exchange rate service unavailable")
	ErrUnsupportedCurrencyPair = errors.New("unsupported currency pair")
	ErrExchangeRateExpired     = errors.New("exchange rate has expired")
	ErrExchangeRateInvalid     = errors.New("invalid exchange rate received")
)

// CurrencyConverter defines the interface for converting amounts between currencies.
type CurrencyConverter interface {
	// Convert converts an amount from one currency to another.
	// Returns the converted amount and the rate used, or an error if conversion is not possible.
	Convert(amount float64, from, to string) (*ConversionInfo, error)

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

// ExchangeRateProvider defines the interface for external exchange rate providers.
type ExchangeRateProvider interface {
	// GetRate fetches the current exchange rate for a currency pair.
	GetRate(from, to string) (*ExchangeRate, error)

	// GetRates fetches multiple exchange rates in a single request.
	GetRates(from string, to []string) (map[string]*ExchangeRate, error)

	// Name returns the provider's name for logging and identification.
	Name() string

	// IsHealthy checks if the provider is currently available.
	IsHealthy() bool
}

// StubCurrencyConverter is a simple implementation that returns the same amount (1:1 conversion).
type StubCurrencyConverter struct {
	rates map[string]map[string]float64
}

// NewStubCurrencyConverter creates a new StubCurrencyConverter with an empty rates map.
func NewStubCurrencyConverter() *StubCurrencyConverter {
	return &StubCurrencyConverter{rates: map[string]map[string]float64{
		"USD": {
			"EUR": 0.84,
			"GBP": 0.76,
			"JPY": 0.0027,
		},
		"EUR": {
			"USD": 1.19,
			"GBP": 0.90,
			"JPY": 0.0024,
		},
		"GBP": {
			"USD": 1.32,
			"EUR": 1.11,
			"JPY": 0.0024,
		},
		"JPY": {
			"USD": 0.0027,
			"EUR": 0.0024,
			"GBP": 0.0024,
		},
	}}
}

func (s *StubCurrencyConverter) Convert(amount float64, from, to string) (*ConversionInfo, error) {
	rate, exists := s.rates[from][to]
	if !exists {
		return nil, ErrUnsupportedCurrencyPair
	}
	return &ConversionInfo{
		OriginalAmount:    amount,
		OriginalCurrency:  from,
		ConvertedAmount:   amount * rate,
		ConvertedCurrency: to,
		ConversionRate:    rate,
	}, nil
}

func (s *StubCurrencyConverter) GetRate(from, to string) (float64, error) {
	if from == to {
		return 1.0, nil
	}
	rate, exists := s.rates[from][to]
	if !exists {
		return 0, ErrUnsupportedCurrencyPair
	}
	return rate, nil
}

func (s *StubCurrencyConverter) IsSupported(from, to string) bool {
	if from == to {
		return true
	}
	_, exists := s.rates[from][to]
	return exists
}
