package service

import (
	"github.com/amirasaad/fintech/pkg/domain"
)

// CurrencyConverter defines the interface for converting amounts between currencies.
type CurrencyConverter interface {
	// Convert converts an amount from one currency to another.
	// Returns the converted amount and the rate used, or an error if conversion is not possible.
	Convert(amount float64, from, to string) (*domain.ConversionInfo, error)
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

func (s *StubCurrencyConverter) Convert(amount float64, from, to string) (*domain.ConversionInfo, error) {
	rate := s.rates[from][to]
	return &domain.ConversionInfo{
		OriginalAmount:    amount,
		OriginalCurrency:  from,
		ConvertedAmount:   amount * rate,
		ConvertedCurrency: to,
		ConversionRate:    rate,
	}, nil
}
