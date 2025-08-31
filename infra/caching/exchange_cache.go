package caching

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/provider/exchange"
	"github.com/amirasaad/fintech/pkg/registry"
)

// ExchangeCache handles bulk caching of exchange rates
// in the infrastructure layer
type ExchangeCache struct {
	exchangeRegistry registry.Provider
	logger           *slog.Logger
	cfg              *config.ExchangeRateCache
	stopChan         chan struct{} // Channel to stop background refresh
}

// NewExchangeCache creates a new ExchangeCache instance
func NewExchangeCache(
	exchangeRegistry registry.Provider,
	logger *slog.Logger,
	cfg *config.ExchangeRateCache,
) *ExchangeCache {
	if cfg == nil {
		cfg = &config.ExchangeRateCache{
			TTL: 15 * time.Minute, // Default TTL if not configured
		}
	}

	if logger == nil {
		logger = slog.Default()
	}

	logger = logger.With(slog.String("component", "exchange_cache"))

	return &ExchangeCache{
		exchangeRegistry: exchangeRegistry,
		logger:           logger,
		cfg:              cfg,
		stopChan:         make(chan struct{}),
	}
}

// GetLastUpdated returns the timestamp of the last rate update
func (c *ExchangeCache) GetLastUpdated(ctx context.Context) (time.Time, error) {
	key := c.getLastUpdatedKey()
	entry, err := c.exchangeRegistry.Get(ctx, key)
	if err != nil {
		// If the entry doesn't exist, return zero time
		if strings.Contains(
			err.Error(), "not found") ||
			strings.Contains(err.Error(), "no such file or directory") ||
			strings.Contains(err.Error(), "entity not found") {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("failed to get last updated time: %w", err)
	}

	if entry == nil {
		return time.Time{}, nil
	}

	// Use reflection to handle different underlying types that might implement the interface
	if info, ok := entry.(interface{ GetTimestamp() time.Time }); ok {
		return info.GetTimestamp(), nil
	}

	// Fallback to checking metadata for timestamp
	if meta, ok := entry.(interface{ Metadata() map[string]string }); ok {
		if tsStr, ok := meta.Metadata()["timestamp"]; ok {
			ts, err := time.Parse(time.RFC3339Nano, tsStr)
			if err == nil {
				return ts, nil
			}
		}
	}

	// If we have an UpdatedAt method, use that
	if updatable, ok := entry.(interface{ UpdatedAt() time.Time }); ok {
		return updatable.UpdatedAt(), nil
	}

	// Last resort: return zero time
	return time.Time{}, nil
}

// getLastUpdatedKey returns the key used to store the last updated timestamp
func (c *ExchangeCache) getLastUpdatedKey() string {
	// Use the exact key format exr:rate:last_updated
	return fmt.Sprintf("%s:last_updated", c.cfg.Prefix)
}

// IsCacheStale checks if the cache is older than the refresh threshold or doesn't exist
// It only checks the last updated timestamp, not individual rate entries
// Returns:
// - bool: true if cache is stale and needs refresh
// - time.Duration: time until next refresh is needed
// - error: any error that occurred
func (c *ExchangeCache) IsCacheStale(ctx context.Context) (bool, time.Duration, error) {
	lastUpdated, err := c.GetLastUpdated(ctx)
	if err != nil {
		// If the entry doesn't exist, consider it stale
		if strings.Contains(err.Error(), "not found") ||
			strings.Contains(err.Error(), "no such file") ||
			strings.Contains(err.Error(), "entity not found") {
			return true, 0, nil
		}
		return false, 0, fmt.Errorf("failed to check cache staleness: %w", err)
	}

	// If we've never updated, cache is stale
	if lastUpdated.IsZero() {
		return true, 0, nil
	}

	// Calculate time since last update
	sinceLastUpdate := time.Since(lastUpdated)

	// Log cache status for debugging
	c.logger.Debug(
		"Cache status check",
		"last_updated", lastUpdated.Format(time.RFC3339),
		"time_since_update", sinceLastUpdate.Round(time.Second).String(),
		"ttl", c.cfg.TTL,
	)

	// Calculate refresh threshold (80% of TTL)
	refreshThreshold := time.Duration(float64(c.cfg.TTL) * 0.8)

	// If we're past the TTL, cache is definitely stale
	if sinceLastUpdate > c.cfg.TTL {
		return true, 0, nil
	}

	// If we're past the refresh threshold, cache is getting stale
	if sinceLastUpdate > refreshThreshold {
		// Cache is getting stale, return time until TTL
		return true, c.cfg.TTL - sinceLastUpdate, nil
	}

	// Cache is still fresh, return time until next refresh
	return false, refreshThreshold - sinceLastUpdate, nil
}

// updateLastUpdated updates the last updated timestamp to now
func (c *ExchangeCache) updateLastUpdated(ctx context.Context) error {
	now := time.Now().UTC()
	key := c.getLastUpdatedKey()
	entry := &exchangeRateInfo{
		BaseEntity: *registry.NewBaseEntity(key, key),
		Timestamp:  now,
	}
	entry.SetActive(true)
	entry.SetMetadata("timestamp", now.Format(time.RFC3339Nano))

	return c.exchangeRegistry.Register(ctx, entry)
}

// CacheRates caches multiple exchange rates in a single operation
func (c *ExchangeCache) CacheRates(
	ctx context.Context,
	rates map[string]*exchange.RateInfo,
	source string,
) error {
	if len(rates) == 0 {
		return nil
	}

	// Create a timestamp that will be used for all cached entries
	now := time.Now().UTC()

	var firstErr error
	// Update the last updated timestamp after all rates are cached
	defer func() {
		if err := c.updateLastUpdated(ctx); err != nil {
			c.logger.Error("Failed to update last updated timestamp", "error", err)
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to update last updated timestamp: %w", err)
			}
		} else {
			c.logger.Debug("Updated exchange rate cache last updated timestamp")
		}
	}()

	count := 0
	for to, rate := range rates {
		if rate == nil {
			c.logger.Error("Skipping nil rate", "to", to)
			continue
		}
		if rate.Rate <= 0 {
			c.logger.Warn(
				"Skipping non-positive conversion rate",
				"from", rate.FromCurrency,
				"to", to,
				"rate", rate.Rate,
			)
			continue
		}

		cacheKey := fmt.Sprintf("exr:rate:%s:%s", rate.FromCurrency, to)
		cacheEntry := &exchangeRateInfo{
			BaseEntity: *registry.NewBaseEntity(cacheKey, cacheKey),
			From:       rate.FromCurrency,
			To:         to,
			Rate:       rate.Rate,
			Source:     source,
			Timestamp:  now,
		}
		cacheEntry.SetActive(true)
		cacheEntry.SetMetadata("source", source)
		cacheEntry.SetMetadata("rate", fmt.Sprintf("%f", rate.Rate))
		cacheEntry.SetMetadata("timestamp", now.Format(time.RFC3339Nano))
		cacheEntry.SetMetadata("from", rate.FromCurrency)
		cacheEntry.SetMetadata("to", to)

		if err := c.exchangeRegistry.Register(ctx, cacheEntry); err != nil {
			c.logger.Error(
				"Failed to cache rate",
				"from", rate.FromCurrency,
				"to", to,
				"error", err,
			)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		count++

		// Cache the inverse rate as well (rate is guaranteed > 0 here)
		inverseRate := 1 / rate.Rate
		inverseKey := fmt.Sprintf("exr:rate:%s:%s", to, rate.FromCurrency)
		inverseEntry := &exchangeRateInfo{
			BaseEntity: *registry.NewBaseEntity(inverseKey, inverseKey),
			From:       to,
			To:         rate.FromCurrency,
			Rate:       inverseRate,
			Source:     source,
			Timestamp:  now,
		}
		inverseEntry.SetActive(true)
		inverseEntry.SetMetadata("source", source)
		inverseEntry.SetMetadata("rate", fmt.Sprintf("%f", inverseRate))
		inverseEntry.SetMetadata(
			"timestamp", now.Format(time.RFC3339Nano),
		)
		inverseEntry.SetMetadata("from", to)
		inverseEntry.SetMetadata("to", rate.FromCurrency)
		inverseEntry.SetMetadata("original_currency", rate.FromCurrency)

		if err := c.exchangeRegistry.Register(ctx, inverseEntry); err != nil {
			c.logger.Error(
				"Failed to cache inverse rate",
				"from", to,
				"to", rate.FromCurrency,
				"error", err,
			)
		} else {
			count++
		}
	}

	// Update last updated timestamp after caching all rates
	lastUpdatedKey := c.getLastUpdatedKey()
	lastUpdatedEntry := &exchangeRateInfo{
		Timestamp: time.Now().UTC(),
	}
	// Set the ID using the proper method
	if err := lastUpdatedEntry.SetID(lastUpdatedKey); err != nil {
		c.logger.Error("Failed to set ID for last updated timestamp",
			"error", err,
			"key", lastUpdatedKey,
		)
	}

	// Update the registry with the last updated timestamp
	if err := c.exchangeRegistry.Register(ctx, lastUpdatedEntry); err != nil {
		c.logger.Error("Failed to update last_updated timestamp",
			"error", err,
			"key", lastUpdatedKey,
		)
		firstErr = fmt.Errorf("failed to update last_updated timestamp: %w", err)
	}

	c.logger.Info("Successfully cached exchange rates",
		"num_rates", count,
		"source", source,
	)

	return firstErr
}

// exchangeRateInfo is a private type used for caching exchange rates and last updated timestamp
// It implements registry.Entity
type exchangeRateInfo struct {
	registry.BaseEntity
	From      string    `json:"from,omitempty"`
	To        string    `json:"to,omitempty"`
	Rate      float64   `json:"rate,omitempty"`
	Source    string    `json:"source,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ID returns the unique identifier for the entity
func (e *exchangeRateInfo) ID() string {
	if id := e.BaseEntity.ID(); id != "" {
		return id
	}
	if e.From != "" && e.To != "" {
		return fmt.Sprintf("exr:rate:%s:%s", e.From, e.To)
	}
	return ""
}

// Name returns the name of the entity
func (e *exchangeRateInfo) Name() string {
	return e.ID()
}

// Active returns whether the entity is active
func (e *exchangeRateInfo) Active() bool {
	return true // Exchange rate entries are always considered active
}

// Metadata returns the metadata for the entity
func (e *exchangeRateInfo) Metadata() map[string]string {
	return map[string]string{
		"from":      e.From,
		"to":        e.To,
		"rate":      fmt.Sprintf("%f", e.Rate),
		"source":    e.Source,
		"timestamp": e.Timestamp.Format(time.RFC3339),
	}
}

// CreatedAt returns the creation time of the entity
func (e *exchangeRateInfo) CreatedAt() time.Time {
	return e.Timestamp
}

// UpdatedAt returns the last update time of the entity
func (e *exchangeRateInfo) UpdatedAt() time.Time {
	return e.Timestamp
}

// Type returns the type of the entity
func (e *exchangeRateInfo) Type() string {
	return "exchange_rate"
}

// GetTimestamp returns the timestamp of the exchange rate
func (e *exchangeRateInfo) GetTimestamp() time.Time {
	return e.Timestamp
}
