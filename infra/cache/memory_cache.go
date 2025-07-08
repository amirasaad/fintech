package cache

import (
	"sync"
	"time"

	"github.com/amirasaad/fintech/pkg/domain"
)

// MemoryCache implements ExchangeRateCache using in-memory storage
type MemoryCache struct {
	cache map[string]*cacheEntry
	mu    sync.RWMutex
}

type cacheEntry struct {
	rate      *domain.ExchangeRate
	expiresAt time.Time
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{
		cache: make(map[string]*cacheEntry),
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a rate from cache
func (c *MemoryCache) Get(key string) (*domain.ExchangeRate, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		return nil, nil
	}

	if time.Now().After(entry.expiresAt) {
		return nil, nil
	}

	return entry.rate, nil
}

// Set stores a rate in cache with TTL
func (c *MemoryCache) Set(key string, rate *domain.ExchangeRate, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = &cacheEntry{
		rate:      rate,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

// Delete removes a rate from cache
func (c *MemoryCache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, key)
	return nil
}

// cleanup removes expired entries from cache
func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.cache {
			if now.After(entry.expiresAt) {
				delete(c.cache, key)
			}
		}
		c.mu.Unlock()
	}
}
