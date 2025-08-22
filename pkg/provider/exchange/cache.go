package exchange

import (
	"context"
	"sync"
	"time"
)

// Cache provides an in-memory cache for exchange rates
type Cache struct {
	store map[string]rateCacheEntry
	mu    sync.RWMutex
	ttl   time.Duration
}

type rateCacheEntry struct {
	value     *RateInfo
	expiresAt time.Time
}

// NewCache creates a new cache with the given TTL
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		store: make(map[string]rateCacheEntry),
		ttl:   ttl,
	}
}

// GetRate gets a rate from the cache
func (c *Cache) GetRate(ctx context.Context, from, to string) (*RateInfo, error) {
	key := cacheKey(from, to)

	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.store[key]
	if !exists {
		return nil, nil
	}

	if time.Now().After(entry.expiresAt) {
		// Entry has expired
		return nil, nil
	}

	return entry.value, nil
}

// StoreRate stores a rate in the cache
func (c *Cache) StoreRate(ctx context.Context, rate *RateInfo) error {
	if rate == nil {
		return nil
	}

	key := cacheKey(rate.FromCurrency, rate.ToCurrency)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.store[key] = rateCacheEntry{
		value:     rate,
		expiresAt: time.Now().Add(c.ttl),
	}

	return nil
}

// BatchGetRates gets multiple rates from the cache
func (c *Cache) BatchGetRates(
	ctx context.Context,
	from string,
	to []string,
) (map[string]*RateInfo, error) {
	result := make(map[string]*RateInfo, len(to))

	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()

	for _, currency := range to {
		key := cacheKey(from, currency)
		if entry, exists := c.store[key]; exists && now.Before(entry.expiresAt) {
			result[currency] = entry.value
		}
	}

	return result, nil
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store = make(map[string]rateCacheEntry)
}

// cacheKey generates a consistent cache key for a currency pair
func cacheKey(from, to string) string {
	return from + "_" + to
}
