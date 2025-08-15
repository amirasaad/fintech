package provider

import (
	"context"
	"errors"
	"time"
)

// Common errors for provider operations
var (
	ErrProviderUnavailable = errors.New("provider unavailable")
	ErrUnsupportedPair     = errors.New("unsupported currency pair")
)

// RateInfo contains information about an exchange rate
// This is the core data structure used by the provider package
type RateInfo struct {
	FromCurrency string    `json:"from_currency"`
	ToCurrency   string    `json:"to_currency"`
	Rate         float64   `json:"rate"`
	Timestamp    time.Time `json:"timestamp"`
	Provider     string    `json:"provider"`
	IsDerived    bool      `json:"is_derived"`
	BaseCurrency string    `json:"base_currency,omitempty"`
	OriginalRate float64   `json:"original_rate,omitempty"`
}

// RateFetcher defines the interface for fetching exchange rates
type RateFetcher interface {
	// FetchRate gets the exchange rate for a currency pair
	FetchRate(ctx context.Context, from, to string) (*RateInfo, error)

	// FetchRates gets multiple exchange rates in a single request
	FetchRates(ctx context.Context, from string, to []string) (map[string]*RateInfo, error)
}

// HealthChecker defines the interface for checking provider health
type HealthChecker interface {
	// CheckHealth checks if the provider is healthy
	CheckHealth(ctx context.Context) error
}

// SupportedChecker defines the interface for checking supported currency pairs
type SupportedChecker interface {
	// IsSupported checks if a currency pair is supported by the provider
	IsSupported(from, to string) bool

	// SupportedPairs returns all supported currency pairs
	SupportedPairs() []string
}

// ProviderMetadata contains metadata about a provider
type ProviderMetadata struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	LastUpdated time.Time `json:"last_updated"`
	IsActive    bool      `json:"is_active"`
}

// Provider defines the complete interface for a rate provider
type Provider interface {
	RateFetcher
	HealthChecker
	SupportedChecker

	// Metadata returns the provider's metadata
	Metadata() ProviderMetadata
}
