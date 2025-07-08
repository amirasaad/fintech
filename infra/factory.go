package infra

import (
	"log/slog"

	"github.com/amirasaad/fintech/infra/cache"
	"github.com/amirasaad/fintech/infra/providers"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/domain"
)

// NewExchangeRateSystem creates a complete exchange rate system with providers, cache, and converter
func NewExchangeRateSystem(logger *slog.Logger, cfg config.ExchangeRateConfig) (domain.CurrencyConverter, error) {
	// Create cache
	rateCache := cache.NewMemoryCache()

	// Create providers
	var exchangeRateProviders []domain.ExchangeRateProvider

	// Add ExchangeRate API provider if API key is configured
	if cfg.ApiKey != "" {
		exchangeRateProvider := providers.NewExchangeRateAPIProvider(cfg.ApiKey, logger)
		exchangeRateProviders = append(exchangeRateProviders, exchangeRateProvider)
		logger.Info("ExchangeRate API provider configured", "apiKey", maskAPIKey(cfg.ApiKey))
	} else {
		logger.Warn("No ExchangeRate API key configured, using fallback only")
	}

	// Create exchange rate service
	exchangeRateService := NewExchangeRateService(exchangeRateProviders, rateCache, logger)

	// Create fallback converter
	var fallback domain.CurrencyConverter
	if cfg.EnableFallback {
		fallback = domain.NewStubCurrencyConverter()
		logger.Info("Fallback currency converter enabled")
	}

	// Create real currency converter
	converter := NewRealCurrencyConverter(exchangeRateService, fallback, logger)

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
