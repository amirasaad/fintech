package service

// CurrencyConverter defines the interface for converting amounts between currencies.
type CurrencyConverter interface {
	// Convert converts an amount from one currency to another.
	// Returns the converted amount and the rate used, or an error if conversion is not possible.
	Convert(amount float64, from, to string) (float64, error)
}

// StubCurrencyConverter is a simple implementation that returns the same amount (1:1 conversion).
type StubCurrencyConverter struct {
	rates map[string]map[string]float64
}

// NewStubCurrencyConverter creates a new StubCurrencyConverter with an empty rates map.
func NewStubCurrencyConverter() *StubCurrencyConverter {
	return &StubCurrencyConverter{rates: make(map[string]map[string]float64)}
}

func (s *StubCurrencyConverter) Convert(amount float64, from, to string) (float64, error) {
	return amount, nil
}
