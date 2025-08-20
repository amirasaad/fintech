package caching

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/amirasaad/fintech/pkg/registry"
)

// ExchangeCache handles bulk caching of exchange rates
// in the infrastructure layer
type ExchangeCache struct {
	exchangeRegistry registry.Provider
	logger           *slog.Logger
	cfg              *config.ExchangeRateCache
}

// NewExchangeCache creates a new ExchangeCache instance
func NewExchangeCache(
	exchangeRegistry registry.Provider,
	logger *slog.Logger,
	cfg *config.ExchangeRateCache,
) *ExchangeCache {
	return &ExchangeCache{
		exchangeRegistry: exchangeRegistry,
		logger:           logger,
		cfg:              cfg,
	}
}

// GetLastUpdated returns the timestamp of the last rate update
func (c *ExchangeCache) GetLastUpdated(ctx context.Context) (time.Time, error) {
	key := c.getLastUpdatedKey()
	entry, err := c.exchangeRegistry.Get(ctx, key)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get last updated time: %w", err)
	}

	if entry == nil {
		return time.Time{}, nil
	}

	switch v := entry.(type) {
	case *exchangeRateInfo:
		return v.Timestamp, nil
	case *registry.BaseEntity:
		// Try to get timestamp from metadata
		if tsStr, ok := v.Metadata()["timestamp"]; ok {
			ts, err := time.Parse(time.RFC3339Nano, tsStr)
			if err == nil {
				return ts, nil
			}
		}
		// Fall back to entity's updated time
		return v.UpdatedAt(), nil
	default:
		return time.Time{}, fmt.Errorf("unexpected entry type: %T", entry)
	}
}

// getLastUpdatedKey returns the key used to store the last updated timestamp
func (c *ExchangeCache) getLastUpdatedKey() string {
	// Use the exact key format exr:rate:last_updated
	return "exr:rate:last_updated"
}

// IsCacheStale checks if the cache is older than the specified TTL or doesn't exist
func (c *ExchangeCache) IsCacheStale(ctx context.Context) (bool, error) {
	lastUpdated, err := c.GetLastUpdated(ctx)
	if err != nil {
		// If the entry doesn't exist, consider the cache stale
		if strings.Contains(err.Error(), "entity not found") {
			return true, nil
		}
		return false, fmt.Errorf("failed to check cache staleness: %w", err)
	}
	return lastUpdated.IsZero() ||
			time.Since(lastUpdated) > c.cfg.TTL,
		nil
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
		cacheEntry.SetActive(true)
		cacheEntry.SetMetadata("source", source)
		cacheEntry.SetMetadata("rate", fmt.Sprintf("%f", rate.ConversionRate))
		cacheEntry.SetMetadata("timestamp", time.Now().UTC().Format(time.RFC3339Nano))
		cacheEntry.SetMetadata("from", rate.OriginalCurrency)
		cacheEntry.SetMetadata("to", to)

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
		inverseEntry.SetActive(true)
		inverseEntry.SetMetadata("source", source)
		inverseEntry.SetMetadata("rate", fmt.Sprintf("%f", inverseRate))
		inverseEntry.SetMetadata("timestamp", time.Now().UTC().Format(time.RFC3339Nano))
		inverseEntry.SetMetadata("from", to)
		inverseEntry.SetMetadata("to", rate.OriginalCurrency)
		inverseEntry.SetMetadata("original_currency", rate.OriginalCurrency)

		if err := c.exchangeRegistry.Register(ctx, inverseEntry); err != nil {
			c.logger.Error("Failed to cache inverse rate",
				"from", to,
				"to", rate.OriginalCurrency,
				"error", err)
		}
	}

	// Update last updated timestamp
	lastUpdatedKey := c.getLastUpdatedKey()
	lastUpdated := &exchangeRateInfo{
		BaseEntity: *registry.NewBaseEntity(lastUpdatedKey, lastUpdatedKey),
		Timestamp:  time.Now().UTC(),
	}
	lastUpdated.SetActive(true)
	lastUpdated.SetMetadata("timestamp", time.Now().UTC().Format(time.RFC3339Nano))

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
