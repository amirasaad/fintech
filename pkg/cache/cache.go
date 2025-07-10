package cache

import (
	"time"

	"github.com/amirasaad/fintech/pkg/domain"
)

// ExchangeRateCache defines the interface for caching exchange rates.
type ExchangeRateCache interface {
	Get(key string) (*domain.ExchangeRate, error)
	Set(key string, rate *domain.ExchangeRate, ttl time.Duration) error
	Delete(key string) error
	GetLastUpdate(key string) (time.Time, error)
	SetLastUpdate(key string, t time.Time) error
}
