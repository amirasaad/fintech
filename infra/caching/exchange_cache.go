package caching

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/amirasaad/fintech/pkg/registry"
)

// ExchangeCache handles bulk caching of exchange rates
// in the infrastructure layer
type ExchangeCache struct {
	exchangeRegistry registry.Provider
	logger           *slog.Logger
	cacheTTL         time.Duration
}

// NewExchangeCache creates a new ExchangeCache instance
func NewExchangeCache(
	exchangeRegistry registry.Provider,
	logger *slog.Logger,
	cacheTTL time.Duration,
) *ExchangeCache {
	return &ExchangeCache{
		exchangeRegistry: exchangeRegistry,
		logger:           logger,
		cacheTTL:         cacheTTL,
	}
}

// CacheRates caches multiple exchange rates in a single operation
func (c *ExchangeCache) CacheRates(
	ctx context.Context,
	rates map[string]*provider.ExchangeInfo,
	source string,
) error {
	if len(rates) == 0 {
		return nil
	}

	var firstErr error
	count := 0
	for to, rate := range rates {
		if rate == nil {
			c.logger.Error("Skipping nil rate", "to", to)
			continue
		}

		// Create cache entry
		cacheKey := fmt.Sprintf("%s:%s", rate.OriginalCurrency, to)
		cacheEntry := &exchangeRateInfo{
			BaseEntity: *registry.NewBaseEntity(cacheKey, cacheKey),
			From:       rate.OriginalCurrency,
			To:         to,
			Rate:       rate.ConversionRate,
			Source:     source,
			Timestamp:  time.Now().UTC(),
		}

		// Save to registry
		if err := c.exchangeRegistry.Register(ctx, cacheEntry); err != nil {
			c.logger.Error("Failed to cache rate",
				"from", rate.OriginalCurrency,
				"to", to,
				"error", err)
			continue
		}

		// Cache the inverse rate as well
		inverseRate := 1 / rate.ConversionRate
		inverseKey := fmt.Sprintf("%s:%s", to, rate.OriginalCurrency)
		inverseEntry := &exchangeRateInfo{
			BaseEntity: *registry.NewBaseEntity(inverseKey, inverseKey),
			From:       to,
			To:         rate.OriginalCurrency,
			Rate:       inverseRate,
			Source:     source,
			Timestamp:  time.Now().UTC(),
		}

		if err := c.exchangeRegistry.Register(ctx, inverseEntry); err != nil {
			c.logger.Error("Failed to cache inverse rate",
				"from", to,
				"to", rate.OriginalCurrency,
				"error", err)
		}
	}

	// Update last updated timestamp
	lastUpdated := &exchangeRateInfo{
		BaseEntity: *registry.NewBaseEntity("exr:last_updated", "exr:last_updated"),
		Timestamp:  time.Now().UTC(),
	}

	if err := c.exchangeRegistry.Register(ctx, lastUpdated); err != nil {
		c.logger.Error("Failed to update last_updated timestamp", "error", err)
		firstErr = fmt.Errorf("failed to update last_updated timestamp: %w", err)
	}

	c.logger.Info("Successfully cached exchange rates",
		"num_rates", count,
		"source", source,
	)

	return firstErr
}

// exchangeRateInfo is a private type used for caching exchange rates
// It implements registry.Entity
// This is a copy of the ExchangeRateInfo from the exchange package
type exchangeRateInfo struct {
	registry.BaseEntity
	From      string
	To        string
	Rate      float64
	Source    string
	Timestamp time.Time
}

// ID returns the unique identifier for the rate info
func (e *exchangeRateInfo) ID() string {
	return e.BaseEntity.ID()
}

// Name returns a human-readable name for the rate info
func (e *exchangeRateInfo) Name() string {
	return e.BaseEntity.Name()
}
