package provider

import (
	"github.com/amirasaad/fintech/pkg/domain"
)

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

func (s *StubCurrencyConverter) Convert(
	amount float64,
	from,
	to string,
) (*domain.ConversionInfo, error) {
	rate, exists := s.rates[from][to]
	if !exists {
		return nil, domain.ErrUnsupportedCurrencyPair
	}
	return &domain.ConversionInfo{
		OriginalAmount:    amount,
		OriginalCurrency:  from,
		ConvertedAmount:   amount * rate,
		ConvertedCurrency: to,
		ConversionRate:    rate,
	}, nil
}

func (s *StubCurrencyConverter) GetRate(
	from,
	to string,
) (float64, error) {
	if from == to {
		return 1.0, nil
	}
	rate, exists := s.rates[from][to]
	if !exists {
		return 0, domain.ErrUnsupportedCurrencyPair
	}
	return rate, nil
}

func (s *StubCurrencyConverter) IsSupported(
	from,
	to string,
) bool {
	if from == to {
		return true
	}
	_, exists := s.rates[from][to]
	return exists
}
