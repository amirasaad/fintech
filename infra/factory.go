package infra

import (
	"log/slog"

	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/amirasaad/fintech/pkg/registry"
	"github.com/amirasaad/fintech/pkg/service/exchange"
)

// NewExchangeRateSystem creates a complete exchange rate system with providers and converter
func NewExchangeRateSystem(
	logger *slog.Logger,
	cfg config.ExchangeRateCache,
	exchangeRateProviders *config.ExchangeRateProviders,
	registryProvider registry.Provider,
) (provider.ExchangeRate, error) {
	if logger == nil {
		logger = slog.Default()
	}

	// Create a default provider (this could be replaced with a real provider)
	// For now, we'll use a simple in-memory provider
	baseProvider := provider.NewBaseProvider("default", "1.0.0", nil)

	// Create exchange rate service with the registry provider and base provider
	exchangeService := exchange.New(
		registryProvider,
		baseProvider, // Pass the base provider
		logger,
	)

	logger.Info("Exchange rate system initialized",
		"providers", 1, // Single provider (the service itself)
		"fallbackEnabled", cfg.EnableFallback,
		"cache_ttl", cfg.TTL.String(),
	)

	// The exchange service implements the provider.ExchangeRate interface
	return exchangeService, nil
}
