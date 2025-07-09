package infra

import (
	"log/slog"

	infra_cache "github.com/amirasaad/fintech/infra/cache"
	infra_provider "github.com/amirasaad/fintech/infra/provider"
	"github.com/amirasaad/fintech/pkg/cache"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/redis/go-redis/v9"
)

// NewExchangeRateSystem creates a complete exchange rate system with providers, cache, and converter
func NewExchangeRateSystem(logger *slog.Logger, cfg config.AppConfig) (domain.CurrencyConverter, error) {
	// Create cache
	var rateCache cache.ExchangeRateCache
	if cfg.Redis.URL != "" {
		opt, err := redis.ParseURL(cfg.Redis.URL)
		if err != nil {
			logger.Error("Invalid Redis URL", "url", cfg.Redis.URL, "error", err)
			return nil, err
		}
		rateCache = infra_cache.NewRedisExchangeRateCacheWithOptions(opt, cfg.Exchange.CachePrefix, logger)
		logger.Info("Using Redis for exchange rate cache", "url", cfg.Redis.URL)
	} else {
		rateCache = infra_cache.NewMemoryCache()
		logger.Info("Using in-memory cache for exchange rates")
	}

	// Create providers
	var exchangeRateProviders []provider.ExchangeRateProvider

	// Use USD as the base currency for now (configurable in future)
	baseCurrency := "USD"
	// TODO: Make base currency configurable via config.Exchange.BaseCurrency

	// Add ExchangeRate API provider if API key is configured
	var exchangeRateProvider *infra_provider.ExchangeRateAPIProvider
	if cfg.Exchange.ApiKey != "" {
		exchangeRateProvider = infra_provider.NewExchangeRateAPIProvider(cfg.Exchange, logger)
		exchangeRateProviders = append(exchangeRateProviders, exchangeRateProvider)
		logger.Info("ExchangeRate API provider configured", "apiKey", maskAPIKey(cfg.Exchange.ApiKey))
	} else {
		logger.Warn("No ExchangeRate API key configured, using fallback only")
	}

	// Fetch and cache rates ONCE at startup for POC
	if exchangeRateProvider != nil {
		err := exchangeRateProvider.FetchAndCacheRates(baseCurrency, rateCache, cfg.Exchange.CacheTTL)
		if err != nil {
			logger.Error("Failed to fetch and cache exchange rates at startup", "error", err)
			// Optionally: return nil, err
		}
	}

	// Create exchange rate service
	exchangeRateService := infra_provider.NewExchangeRateService(exchangeRateProviders, rateCache, logger)

	// Create fallback converter
	var fallback domain.CurrencyConverter
	if cfg.Exchange.EnableFallback {
		fallback = infra_provider.NewStubCurrencyConverter()
		logger.Info("Fallback currency converter enabled")
	}

	// Create real currency converter
	converter := infra_provider.NewRealCurrencyConverter(exchangeRateService, fallback, logger)

	logger.Info("Exchange rate system initialized",
		"providers", len(exchangeRateProviders),
		"fallbackEnabled", cfg.Exchange.EnableFallback,
		"cacheTTL", cfg.Exchange.CacheTTL)

	return converter, nil
}

// maskAPIKey returns a masked version of the API key for logging
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
}
