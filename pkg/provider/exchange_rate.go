package provider

import "github.com/amirasaad/fintech/pkg/domain"

// ExchangeRate defines the interface for external exchange rate providers.
type ExchangeRate interface {
	// GetRate fetches the current exchange rate for a currency pair.
	GetRate(from, to string) (*domain.ExchangeRate, error)

	// GetRates fetches multiple exchange rates in a single request.
	GetRates(from string, to []string) (map[string]*domain.ExchangeRate, error)

	// Name returns the provider's name for logging and identification.
	Name() string

	// IsHealthy checks if the provider is currently available.
	IsHealthy() bool
}
