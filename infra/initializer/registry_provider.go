package initializer

import (
	"log/slog"
	"time"

	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/registry"
)

// RegistryConfig holds configuration for creating a registry provider
type RegistryConfig struct {
	Name      string
	RedisURL  string
	KeyPrefix string
	CacheSize int
	CacheTTL  time.Duration
}

// GetRegistryProvider returns a configured registry provider based on the provided config
func GetRegistryProvider(
	cfg *RegistryConfig,
	logger *slog.Logger,
) (registry.Provider, error) {
	if cfg == nil {
		cfg = &RegistryConfig{
			Name:      "default",
			CacheSize: 1000,
			CacheTTL:  -1, // No expiration
		}
	}

	// Ensure cache size is at least 1 if caching is enabled
	if cfg.CacheTTL != 0 && cfg.CacheSize <= 0 {
		cfg.CacheSize = 1000
	}

	builder := registry.NewBuilder().
		WithName(cfg.Name).
		WithRedis(cfg.RedisURL).
		WithKeyPrefix(cfg.KeyPrefix).
		WithCache(cfg.CacheSize, cfg.CacheTTL)

	logger.Info("Creating registry provider",
		"name", cfg.Name,
		"redis_configured", cfg.RedisURL != "",
		"key_prefix", cfg.KeyPrefix,
		"cache_size", cfg.CacheSize,
		"cache_ttl", cfg.CacheTTL,
	)

	return builder.BuildRegistry()
}

// GetCheckoutRegistry creates a registry provider for the checkout service
func GetCheckoutRegistry(cfg *config.App, logger *slog.Logger) (registry.Provider, error) {
	keyPrefix := ""
	if cfg.Redis != nil {
		keyPrefix = cfg.Redis.KeyPrefix
	}

	registryCfg := &RegistryConfig{
		Name:      "checkout",
		RedisURL:  cfg.Redis.URL,
		KeyPrefix: keyPrefix + "checkout:",
		CacheSize: 1000,
		CacheTTL:  -1, // No expiration for checkout sessions
	}

	return GetRegistryProvider(registryCfg, logger)
}

// GetExchangeRateRegistry creates a registry provider for the exchange rate service
func GetExchangeRateRegistry(cfg *config.App, logger *slog.Logger) (registry.Provider, error) {
	if cfg.ExchangeRateCache == nil {
		return nil, nil
	}

	keyPrefix := cfg.ExchangeRateCache.Prefix
	if keyPrefix == "" {
		keyPrefix = "exr:rate:"
	}

	registryCfg := &RegistryConfig{
		Name:      "exchange_rate",
		RedisURL:  cfg.ExchangeRateCache.Url,
		KeyPrefix: keyPrefix,
		CacheSize: 1000,
		CacheTTL:  cfg.ExchangeRateCache.TTL,
	}

	return GetRegistryProvider(registryCfg, logger)
}

// GetDefaultRegistry creates a default registry provider
func GetDefaultRegistry(cfg *config.App, logger *slog.Logger) (registry.Provider, error) {
	keyPrefix := ""
	if cfg.Redis != nil {
		keyPrefix = cfg.Redis.KeyPrefix
	}

	registryCfg := &RegistryConfig{
		Name:      "default",
		RedisURL:  cfg.Redis.URL,
		KeyPrefix: keyPrefix,
		CacheSize: 1000,
		CacheTTL:  -1, // No expiration
	}

	return GetRegistryProvider(registryCfg, logger)
}
