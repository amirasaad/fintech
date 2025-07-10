package provider

import (
	"fmt"
	"log/slog"
	"math"
	"time"

	infra_cache "github.com/amirasaad/fintech/infra/cache"
	"github.com/amirasaad/fintech/pkg/cache"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/provider"
)

// ExchangeRateService provides real-time exchange rates with caching and fallback providers.
type ExchangeRateService struct {
	providers []provider.ExchangeRateProvider
	cache     cache.ExchangeRateCache
	logger    *slog.Logger
	cfg       *config.ExchangeRateConfig
	// mu        sync.RWMutex
}

// NewExchangeRateService creates a new exchange rate service with the given providers, cache, and exchange rate config.
func NewExchangeRateService(providers []provider.ExchangeRateProvider, cache cache.ExchangeRateCache, logger *slog.Logger, cfg *config.ExchangeRateConfig) *ExchangeRateService {
	return &ExchangeRateService{
		providers: providers,
		cache:     cache,
		logger:    logger,
		cfg:       cfg,
	}
}

// GetRate retrieves an exchange rate, trying cache first, then providers in order.
func (s *ExchangeRateService) GetRate(from, to string) (*domain.ExchangeRate, error) {
	s.logger.Info("[DIAG] Cache type", "type", fmt.Sprintf("%T", s.cache))
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

	cacheKey := fmt.Sprintf("%s:%s", from, to)

	// Check last update timestamp
	if redisCache, ok := s.cache.(*infra_cache.RedisExchangeRateCache); ok {
		s.logger.Info("[DIAG] Checking last update timestamp in Redis", "key", cacheKey)
		lastUpdate, err := redisCache.GetLastUpdate(cacheKey)
		if err != nil {
			s.logger.Warn("[DIAG] Failed to get last update from Redis", "key", cacheKey, "error", err)
		}
		if err == nil && !lastUpdate.IsZero() {
			s.logger.Info("[DIAG] Last update timestamp found", "key", cacheKey, "lastUpdate", lastUpdate, "age", time.Since(lastUpdate), "ttl", s.cfg.CacheTTL)
		}
		if err == nil && !lastUpdate.IsZero() && time.Since(lastUpdate) < s.cfg.CacheTTL {
			// Try cache first
			if cached, err := s.cache.Get(cacheKey); err == nil && cached != nil {
				if time.Now().Before(cached.ExpiresAt) {
					s.logger.Info("[DIAG] Exchange rate retrieved from cache (last update valid)", "from", from, "to", to, "rate", cached.Rate)
					return cached, nil
				}
			}
		}
	}

	// Try reverse pair in cache and invert
	reverseKey := fmt.Sprintf("%s:%s", to, from)
	if cached, err := s.cache.Get(reverseKey); err == nil && cached != nil {
		if time.Now().Before(cached.ExpiresAt) && cached.Rate != 0 {
			s.logger.Debug("Exchange rate retrieved from cache (reversed)", "from", to, "to", from, "rate", cached.Rate)
			return &domain.ExchangeRate{
				FromCurrency: from,
				ToCurrency:   to,
				Rate:         1 / cached.Rate,
				LastUpdated:  cached.LastUpdated,
				Source:       cached.Source + " (reversed)",
				ExpiresAt:    cached.ExpiresAt,
			}, nil
		}
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
			if redisCache, ok := s.cache.(*infra_cache.RedisExchangeRateCache); ok {
				_ = redisCache.SetLastUpdate(cacheKey, time.Now())
			}
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
