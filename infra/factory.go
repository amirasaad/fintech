package infra

import (
	"log/slog"

	infra_cache "github.com/amirasaad/fintech/infra/cache"
	infra_provider "github.com/amirasaad/fintech/infra/provider"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/provider"
)

// NewExchangeRateSystem creates a complete exchange rate system with providers, cache, and converter
func NewExchangeRateSystem(logger *slog.Logger, cfg config.ExchangeRateConfig) (domain.CurrencyConverter, error) {
	// Create cache
	rateCache := infra_cache.NewMemoryCache()

	// Create providers
	var exchangeRateProviders []provider.ExchangeRateProvider

	// Add ExchangeRate API provider if API key is configured
	if cfg.ApiKey != "" {
		exchangeRateProvider := infra_provider.NewExchangeRateAPIProvider(cfg.ApiKey, logger)
		exchangeRateProviders = append(exchangeRateProviders, exchangeRateProvider)
		logger.Info("ExchangeRate API provider configured", "apiKey", maskAPIKey(cfg.ApiKey))
	} else {
		logger.Warn("No ExchangeRate API key configured, using fallback only")
	}

	// Create exchange rate service
	exchangeRateService := infra_provider.NewExchangeRateService(exchangeRateProviders, rateCache, logger)

	// Create fallback converter
	var fallback domain.CurrencyConverter
	if cfg.EnableFallback {
		fallback = infra_provider.NewStubCurrencyConverter()
		logger.Info("Fallback currency converter enabled")
	}

	// Create real currency converter
	converter := infra_provider.NewRealCurrencyConverter(exchangeRateService, fallback, logger)

	logger.Info("Exchange rate system initialized",
		"providers", len(exchangeRateProviders),
		"fallbackEnabled", cfg.EnableFallback,
		"cacheTTL", cfg.CacheTTL)

	return converter, nil
}

// maskAPIKey returns a masked version of the API key for logging
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
}
