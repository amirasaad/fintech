package infra

import (
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/amirasaad/fintech/pkg/domain"
)

// ExchangeRateService provides real-time exchange rates with caching and fallback providers.
type ExchangeRateService struct {
	providers []domain.ExchangeRateProvider
	cache     ExchangeRateCache
	logger    *slog.Logger
	mu        sync.RWMutex
}

// ExchangeRateCache defines the interface for caching exchange rates.
type ExchangeRateCache interface {
	Get(key string) (*domain.ExchangeRate, error)
	Set(key string, rate *domain.ExchangeRate, ttl time.Duration) error
	Delete(key string) error
}

// NewExchangeRateService creates a new exchange rate service with the given providers and cache.
func NewExchangeRateService(providers []domain.ExchangeRateProvider, cache ExchangeRateCache, logger *slog.Logger) *ExchangeRateService {
	return &ExchangeRateService{
		providers: providers,
		cache:     cache,
		logger:    logger,
	}
}

// GetRate retrieves an exchange rate, trying cache first, then providers in order.
func (s *ExchangeRateService) GetRate(from, to string) (*domain.ExchangeRate, error) {
	if from == to {
		return &domain.ExchangeRate{
			FromCurrency: from,
			ToCurrency:   to,
			Rate:         1.0,
			LastUpdated:  time.Now(),
			Source:       "internal",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		}, nil
	}

	// Try cache first
	cacheKey := fmt.Sprintf("%s:%s", from, to)
	if cached, err := s.cache.Get(cacheKey); err == nil && cached != nil {
		if time.Now().Before(cached.ExpiresAt) {
			s.logger.Debug("Exchange rate retrieved from cache", "from", from, "to", to, "rate", cached.Rate)
			return cached, nil
		}
		// Rate expired, remove from cache
		_ = s.cache.Delete(cacheKey)
	}

	// Try providers in order
	for _, provider := range s.providers {
		if !provider.IsHealthy() {
			s.logger.Warn("Provider not healthy, skipping", "provider", provider.Name())
			continue
		}

		rate, err := provider.GetRate(from, to)
		if err != nil {
			s.logger.Warn("Failed to get rate from provider", "provider", provider.Name(), "error", err)
			continue
		}

		// Validate rate
		if rate.Rate <= 0 || math.IsNaN(rate.Rate) || math.IsInf(rate.Rate, 0) {
			s.logger.Warn("Invalid rate received from provider", "provider", provider.Name(), "rate", rate.Rate)
			continue
		}

		// Cache the rate
		ttl := time.Until(rate.ExpiresAt)
		if ttl > 0 {
			_ = s.cache.Set(cacheKey, rate, ttl)
		}

		s.logger.Info("Exchange rate retrieved from provider", "provider", provider.Name(), "from", from, "to", to, "rate", rate.Rate)
		return rate, nil
	}

	return nil, domain.ErrExchangeRateUnavailable
}

// GetRates retrieves multiple exchange rates efficiently.
func (s *ExchangeRateService) GetRates(from string, to []string) (map[string]*domain.ExchangeRate, error) {
	results := make(map[string]*domain.ExchangeRate)
	var missing []string

	// Try cache first for each currency
	for _, currency := range to {
		cacheKey := fmt.Sprintf("%s:%s", from, currency)
		if cached, err := s.cache.Get(cacheKey); err == nil && cached != nil {
			if time.Now().Before(cached.ExpiresAt) {
				results[currency] = cached
				continue
			}
			_ = s.cache.Delete(cacheKey)
		}
		missing = append(missing, currency)
	}

	// If we have missing rates, try providers
	if len(missing) > 0 {
		for _, provider := range s.providers {
			if !provider.IsHealthy() {
				continue
			}

			rates, err := provider.GetRates(from, missing)
			if err != nil {
				s.logger.Warn("Failed to get rates from provider", "provider", provider.Name(), "error", err)
				continue
			}

			// Cache and add valid rates
			for currency, rate := range rates {
				if rate.Rate > 0 && !math.IsNaN(rate.Rate) && !math.IsInf(rate.Rate, 0) {
					results[currency] = rate
					cacheKey := fmt.Sprintf("%s:%s", from, currency)
					ttl := time.Until(rate.ExpiresAt)
					if ttl > 0 {
						_ = s.cache.Set(cacheKey, rate, ttl)
					}
				}
			}

			// Update missing list
			missing = missing[:0]
			for _, currency := range to {
				if _, exists := results[currency]; !exists {
					missing = append(missing, currency)
				}
			}

			if len(missing) == 0 {
				break
			}
		}
	}

	if len(results) == 0 {
		return nil, domain.ErrExchangeRateUnavailable
	}

	return results, nil
}
