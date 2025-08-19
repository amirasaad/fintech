package initializer

import (
	"log/slog"

	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/registry"
)

// GetRegistryProvider returns a configured registry provider for the given service
func GetRegistryProvider(
	serviceName string,
	cfg *config.App,
	logger *slog.Logger,
) (registry.Provider, error) {
	// Special handling for checkout service
	if serviceName == "checkout" {
		builder := registry.NewBuilder().
			WithName("checkout").
			WithRedis(cfg.Redis.URL).
			WithKeyPrefix("checkout:").
			WithCache(1000, -1)

		logger.Info("Using checkout registry config",
			"redis_configured", cfg.Redis.URL != "",
			"key_prefix", "checkout:")

		return builder.BuildRegistry()
	}

	// Special handling for exchange rate service
	if serviceName == "exchange_rate" && cfg.ExchangeRateCache != nil {
		prefix := cfg.ExchangeRateCache.Prefix
		if prefix == "" {
			prefix = "exr:rate:" // Default prefix for exchange rates
		}

		builder := registry.NewBuilder().
			WithName("exchange_rate").
			WithRedis(cfg.ExchangeRateCache.Url).
			WithKeyPrefix(prefix).
			WithCache(1000, cfg.ExchangeRateCache.TTL)

		logger.Info("Using exchange rate specific registry config",
			"redis_configured", cfg.ExchangeRateCache.Url != "",
			"key_prefix", prefix,
			"ttl", cfg.ExchangeRateCache.TTL)

		return builder.BuildRegistry()
	}

	// Default registry configuration for other services
	builder := registry.NewBuilder().
		WithName(serviceName).
		WithRedis(cfg.Redis.URL).
		WithKeyPrefix(cfg.Redis.KeyPrefix+serviceName+":").
		WithCache(1000, -1)

	// Build and return the registry
	provider, err := builder.BuildRegistry()
	if err != nil {
		logger.Error("Failed to create registry provider",
			"service", serviceName,
			"error", err)
		return nil, err
	}

	logger.Info("Initialized registry provider",
		"service", serviceName,
		"redis_configured", cfg.Redis.URL != "")

	return provider, nil
}
