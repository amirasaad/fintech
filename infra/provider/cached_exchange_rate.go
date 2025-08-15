package provider

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/amirasaad/fintech/pkg/provider"
)

// ExchangeRateCache defines the interface for caching exchange rates.
type ExchangeRateCache interface {
	Get(key string) (*provider.ExchangeInfo, error)
	Set(key string, rate *provider.ExchangeInfo, ttl time.Duration) error
	Delete(key string) error
	GetLastUpdate(key string) (time.Time, error)
	SetLastUpdate(key string, t time.Time) error
}

// CachedExchangeRate implements ExchangeRate with caching capabilities.
type CachedExchangeRate struct {
	next   provider.ExchangeRate
	cache  ExchangeRateCache
	ttl    time.Duration
	logger *slog.Logger
}

// NewCachedExchangeRate creates a new CachedExchangeRate.
func NewCachedExchangeRate(
	next provider.ExchangeRate,
	cache ExchangeRateCache,
	ttl time.Duration,
	logger *slog.Logger,
) *CachedExchangeRate {
	return &CachedExchangeRate{
		next:   next,
		cache:  cache,
		ttl:    ttl,
		logger: logger,
	}
}

// GetRate fetches the current exchange rate for a currency pair, using cache.
func (c *CachedExchangeRate) GetRate(
	ctx context.Context,
	from, to string,
) (*provider.ExchangeInfo, error) {
	key := fmt.Sprintf("exchange_rate:%s-%s", from, to)

	// Try to get from cache
	if rate, err := c.cache.Get(key); err == nil && rate != nil {
		c.logger.Debug("Cache hit for GetRate", "key", key)
		return rate, nil
	} else if err != nil {
		c.logger.Error("Error getting from cache", "key", key, "error", err)
	}

	c.logger.Debug("Cache miss for GetRate, fetching from next provider", "key", key)

	// Fetch from next provider
	rate, err := c.next.GetRate(ctx, from, to)
	if err != nil {
		return nil, err
	}

	// Store in cache
	if err := c.cache.Set(key, rate, c.ttl); err != nil {
		c.logger.Error("Error setting cache for GetRate", "key", key, "error", err)
	}

	return rate, nil
}

// GetRates fetches multiple exchange rates, using cache.
func (c *CachedExchangeRate) GetRates(
	ctx context.Context,
	from string,
	to []string,
) (map[string]*provider.ExchangeInfo, error) {
	results := make(map[string]*provider.ExchangeInfo)
	var toFetch []string

	// Try to get from cache for each currency pair
	for _, t := range to {
		key := fmt.Sprintf("exchange_rate:%s-%s", from, t)
		if rate, err := c.cache.Get(key); err == nil && rate != nil {
			c.logger.Debug("Cache hit for GetRates", "key", key)
			results[t] = rate
		} else if err != nil {
			c.logger.Error("Error getting from cache", "key", key, "error", err)
			toFetch = append(toFetch, t) // If error, try to fetch
		} else {
			c.logger.Debug("Cache miss for GetRates", "key", key)
			toFetch = append(toFetch, t)
		}
	}

	if len(toFetch) > 0 {
		c.logger.Debug("Fetching missing rates from next provider", "currencies", toFetch)
		fetchedRates, err := c.next.GetRates(ctx, from, toFetch)
		if err != nil {
			return nil, err
		}

		for t, rate := range fetchedRates {
			key := fmt.Sprintf("exchange_rate:%s-%s", from, t)
			results[t] = rate
			// Store in cache
			if err := c.cache.Set(key, rate, c.ttl); err != nil {
				c.logger.Error("Error setting cache for GetRates", "key", key, "error", err)
			}
		}
	}

	return results, nil
}

// IsSupported checks if a currency pair is supported by the underlying provider.
func (c *CachedExchangeRate) IsSupported(from, to string) bool {
	return c.next.IsSupported(from, to)
}

// Name returns the provider's name.
func (c *CachedExchangeRate) Name() string {
	return fmt.Sprintf("Cached(%s)", c.next.Name())
}

// IsHealthy checks if the underlying provider is healthy.
func (c *CachedExchangeRate) IsHealthy() bool {
	return c.next.IsHealthy()
}

// Ensure CachedExchangeRate implements provider.ExchangeRate
var _ provider.ExchangeRate = (*CachedExchangeRate)(nil)
