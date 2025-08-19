package exchange

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/amirasaad/fintech/infra/caching"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/amirasaad/fintech/pkg/registry"
)

// ---- Errors ----

var (
	ErrInvalidAmount        = errors.New("invalid amount")
	ErrNoProvidersAvailable = errors.New("no exchange rate providers available")
	ErrInvalidExchangeRate  = errors.New("invalid exchange rate")
)

// ---- Constants ----

const (
	// DefaultCacheTTL is the default time-to-live for cached exchange rates
	DefaultCacheTTL = 15 * time.Minute
	// LastUpdatedKey is the key used to store the last update timestamp
	LastUpdatedKey = "exr:last_updated"
)

// ---- Entity ----

type ExchangeRateInfo struct {
	registry.BaseEntity
	From      string
	To        string
	Rate      float64
	Source    string
	Timestamp time.Time
}

// newExchangeRateInfo constructs an ExchangeRateInfo with initialized BaseEntity (ID and Name)
// to satisfy registry validation requirements and ensure proper caching behavior.
func newExchangeRateInfo(
	from, to string,
	rate float64,
	source string,
) *ExchangeRateInfo {
	id := fmt.Sprintf("%s:%s", from, to)
	return &ExchangeRateInfo{
		BaseEntity: *registry.NewBaseEntity(id, id),
		From:       from,
		To:         to,
		Rate:       rate,
		Source:     source,
		Timestamp:  time.Now().UTC(),
	}
}

// ---- Conversion Helpers ----

// ---- Helper Functions ----

func identityRate(from, to string) *provider.ExchangeInfo {
	return &provider.ExchangeInfo{
		OriginalCurrency:  from,
		ConvertedCurrency: to,
		ConversionRate:    1.0,
		Source:            "identity",
		Timestamp:         time.Now(),
	}
}

func validateAmount(amount *money.Money) error {
	if amount == nil {
		return errors.New("amount cannot be nil")
	}
	if amount.IsNegative() || amount.IsZero() {
		return ErrInvalidAmount
	}
	return nil
}

// ---- Service ----

// Service handles currency exchange operations
type Service struct {
	provider         provider.ExchangeRate
	logger           *slog.Logger
	exchangeRegistry registry.Provider      // Specific registry for exchange rates
	cacheTTL         time.Duration          // TTL for cached rates
	exchangeCache    *caching.ExchangeCache // Handles bulk caching operations
}

// New creates a new exchange service with the given exchange registry, provider, and logger.
func New(
	exchangeRegistry registry.Provider,
	provider provider.ExchangeRate,
	log *slog.Logger,
) *Service {
	if log == nil {
		log = slog.Default()
	}

	s := &Service{
		provider:         provider,
		logger:           log,
		exchangeRegistry: exchangeRegistry,
		cacheTTL:         DefaultCacheTTL,
	}

	// Initialize the exchange cache with a logger adapter
	s.exchangeCache = caching.NewExchangeCache(exchangeRegistry, log, DefaultCacheTTL)

	return s
}

// areRatesCached checks if we have valid cached rates for all supported currency pairs
func (s *Service) areRatesCached(ctx context.Context, from string) (bool, error) {
	// Get provider name for logging
	providerName := s.provider.(interface{ Name() string }).Name()
	s.logger.Debug("Checking for cached rates", "from", from, "provider", providerName)

	// Check if provider supports getting all rates at once
	if getRatesProvider, ok := s.provider.(interface {
		GetRates(ctx context.Context, from string) (map[string]*provider.ExchangeInfo, error)
	}); ok {
		// Get list of all supported target currencies
		rates, err := getRatesProvider.GetRates(ctx, from)
		if err != nil {
			s.logger.Warn("Failed to get supported currencies from provider",
				"from", from, "provider", providerName, "error", err)
			return false, err
		}

		// If no rates are supported, we can't cache anything
		if len(rates) == 0 {
			s.logger.Debug("No rates available from provider",
				"from", from,
				"provider", providerName)
			return false, nil
		}

		// Check if we have valid cached rates for all supported currencies
		for to := range rates {
			cacheKey := fmt.Sprintf("%s:%s", from, to)
			entity, err := s.exchangeRegistry.Get(ctx, cacheKey)
			if err != nil || entity == nil {
				s.logger.Debug("Rate not found in cache", "from", from, "to", to)
				return false, nil
			}

			// Check if the cached rate is still valid
			if _, ok := s.getValidCachedRate(entity, from, to, 24*time.Hour); !ok {
				s.logger.Debug("Cached rate expired or invalid", "from", from, "to", to)
				return false, nil
			}
		}

		s.logger.Info("All rates found in cache and are valid",
			"from", from, "provider", providerName, "num_rates", len(rates))
		return true, nil
	}

	// If provider doesn't support GetRates, we can't check all rates at once
	s.logger.Debug("Provider does not support getting all rates, can't check cache status")
	return false, nil
}

// shouldFetchNewRates checks if we should fetch new rates based on last_updated timestamp
func (s *Service) shouldFetchNewRates(ctx context.Context) (bool, error) {
	// Try to get last_updated timestamp
	entity, err := s.exchangeRegistry.Get(ctx, LastUpdatedKey)
	if err != nil {
		s.logger.Debug("No last_updated timestamp found, will fetch new rates",
			"key", LastUpdatedKey, "error", err)
		return true, nil
	}

	if entity == nil {
		s.logger.Debug("No last_updated timestamp found, will fetch new rates",
			"key", LastUpdatedKey)
		return true, nil
	}

	info, ok := entity.(*ExchangeRateInfo)
	if !ok {
		s.logger.Warn("Invalid last_updated entity type, will fetch new rates",
			"type", fmt.Sprintf("%T", entity))
		return true, nil
	}

	// Check if cache is still valid
	if time.Since(info.Timestamp) < s.cacheTTL {
		s.logger.Debug("Using cached rates, last updated",
			"last_updated", info.Timestamp, "ttl", s.cacheTTL)
		return false, nil
	}

	s.logger.Info("Cache expired, fetching new rates",
		"last_updated", info.Timestamp, "ttl", s.cacheTTL)
	return true, nil
}

// FetchAndCacheRates fetches and caches exchange rates for the given currency
// It checks the last_updated timestamp to determine if we need to fetch new rates
func (s *Service) FetchAndCacheRates(
	ctx context.Context,
	from string,
) error {
	// Check if we need to fetch new rates based on last_updated
	shouldFetch, err := s.shouldFetchNewRates(ctx)
	if err != nil {
		s.logger.Warn("Error checking last_updated, will proceed with fetch",
			"from", from, "error", err)
	} else if !shouldFetch {
		// We have fresh rates, check if we have rates for the requested currency
		cached, err := s.areRatesCached(ctx, from)
		if err == nil && cached {
			s.logger.Info("Rates already cached and fresh, skipping fetch",
				"from", from)
			return nil
		}
	}

	// Validate provider health
	if err := s.validateProviderHealth(); err != nil {
		s.logger.Warn("Skipping rate fetch due to unhealthy provider",
			"from", from, "error", err)
		return err
	}

	// Get provider name for logging
	providerName := s.provider.(interface{ Name() string }).Name()

	// Get rates from provider
	rates, err := s.provider.GetRates(ctx, from)
	if err != nil {
		s.logger.Warn(
			"Failed to get rates from provider",
			"provider", providerName,
			"error", err,
		)
		return err
	}

	// Cache all rates at once using the exchange cache
	if err := s.exchangeCache.CacheRates(ctx, rates, providerName); err != nil {
		s.logger.Error("Failed to cache exchange rates",
			"from", from,
			"num_rates", len(rates),
			"error", err)
		return fmt.Errorf("failed to cache exchange rates: %w", err)
	}

	s.logger.Info("Successfully fetched and cached exchange rates",
		"from", from, "num_rates", len(rates), "provider", providerName)
	return nil
}

// validateProviderHealth checks provider presence and health.
func (s *Service) validateProviderHealth() error {
	if s.provider == nil {
		s.logger.Error(
			"Failed to get active providers",
			"error", "no exchange rate providers available")
		return fmt.Errorf("no exchange rate providers available")
	}
	healthy, name := true, ""
	if p, ok := s.provider.(interface{ IsHealthy() bool }); ok {
		healthy = p.IsHealthy()
	}
	if p, ok := s.provider.(interface{ Name() string }); ok {
		name = p.Name()
	}
	if !healthy {
		s.logger.Warn(
			"Skipping unhealthy provider",
			"provider", name,
		)
		return fmt.Errorf("provider %s is unhealthy", name)
	}
	return nil
}

// processAndCacheRate validates, logs, and caches a rate.
// It uses the exchange cache to handle the actual caching.
// This is a convenience method that wraps the bulk caching functionality
// for a single rate.
func (s *Service) processAndCacheRate(from string, to string, rate *provider.ExchangeInfo) {
	if rate == nil {
		err := fmt.Errorf("provider %s returned nil rate", s.provider.Name())
		s.logger.Error("Failed to fetch exchange rate",
			"from", from,
			"to", to,
			"provider", s.provider.Name(),
			"error", err,
		)
		return
	}

	// Create a rates map with a single rate
	rates := map[string]*provider.ExchangeInfo{
		to: rate,
	}

	// Use the exchange cache to handle the caching
	if err := s.exchangeCache.CacheRates(
		context.Background(),
		rates,
		s.provider.Name(),
	); err != nil {
		s.logger.Error("Failed to cache exchange rate",
			"from", from,
			"to", to,
			"error", err,
		)
	}
}

// ---- Public Service Methods ----

func (s *Service) Name() string    { return "ExchangeService" }
func (s *Service) IsHealthy() bool { return true }

// Convert converts an amount from one currency to another.
// It first checks the cache for a valid rate, and if not found, fetches it from the provider.
func (s *Service) Convert(
	ctx context.Context,
	amount *money.Money,
	to money.Code,
) (*money.Money, *provider.ExchangeInfo, error) {
	if err := validateAmount(amount); err != nil {
		return nil, nil, fmt.Errorf("invalid amount: %w", err)
	}

	from := amount.Currency().String()
	toStr := to.String()

	// Check if conversion is needed
	if from == toStr {
		return amount, &provider.ExchangeInfo{
			OriginalCurrency:  from,
			ConvertedCurrency: toStr,
			ConversionRate:    1.0,
			Source:            "identity",
			Timestamp:         time.Now(),
		}, nil
	}

	// Try to get rate from cache first
	rate, err := s.GetRate(ctx, from, toStr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get exchange rate: %w", err)
	}

	// Convert the amount
	converted, err := amount.Multiply(rate.ConversionRate)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert amount: %w", err)
	}

	result, err := money.New(converted.AmountFloat(), to)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create money: %w", err)
	}

	return result, rate, nil
}

func (s *Service) GetRate(ctx context.Context, from, to string) (*provider.ExchangeInfo, error) {
	return s.getRateHelper(ctx, from, to)
}

// getRateHelper contains the GetRate logic.
func (s *Service) getRateHelper(
	ctx context.Context,
	from, to string,
) (*provider.ExchangeInfo, error) {
	if from == to {
		return identityRate(from, to), nil
	}

	// First try to get from cache
	if cached, ok := s.getRateFromCache(ctx, from, to); ok {
		return cached, nil
	}

	// If not in cache, fetch from provider
	rate, err := s.provider.GetRate(ctx, from, to)
	if err != nil {
		s.logger.Warn("Failed to get rate from provider",
			"from", from, "to", to, "error", err)
		return nil, fmt.Errorf("failed to get rate from provider: %w", err)
	}

	// Validate the fetched rate
	if err := validateFetchedRate(rate); err != nil {
		s.logger.Warn("Invalid rate from provider",
			"from", from, "to", to, "rate", rate.ConversionRate)
		return nil, fmt.Errorf("invalid rate from provider: %w", err)
	}

	// Save the rate to cache
	rateInfo := newExchangeRateInfo(from, to, rate.ConversionRate, s.provider.Name())
	s.logger.Info("Successfully fetched exchange rate",
		"from", from, "to", to, "rate", rate.ConversionRate)

	// Save to registry
	s.saveDirectAndInverseRates(ctx, from, to, rateInfo)
	return rate, nil
}

// GetRates fetches multiple exchange rates in a single request.
// Implements provider.ExchangeRate interface.
func (s *Service) GetRates(
	ctx context.Context,
	from string,
) (map[string]*provider.ExchangeInfo, error) {
	// First try to get all rates from cache
	cached, err := s.areRatesCached(ctx, from)
	if err == nil && cached {
		// If all rates are cached, return them
		rates := make(map[string]*provider.ExchangeInfo)
		// Get all supported target currencies
		targetRates, err := s.provider.GetRates(ctx, from)
		if err != nil {
			s.logger.Warn("Failed to get supported currencies from provider",
				"from", from, "error", err)
			return nil, err
		}

		for to := range targetRates {
			cachedRate, ok := s.getRateFromCache(ctx, from, to)
			if !ok {
				s.logger.Warn("Failed to get rate from cache",
					"from", from, "to", to)
				continue
			}
			rates[to] = cachedRate
		}

		s.logger.Info("Returning all rates from cache", "from", from, "count", len(rates))
		return rates, nil
	}

	// If not all rates are cached or there was an error checking the cache,
	// fetch them from the provider
	rates, err := s.provider.GetRates(ctx, from)
	if err != nil {
		s.logger.Warn("Failed to get rates from provider",
			"from", from, "error", err)
		return nil, err
	}

	// Cache the rates
	for to, rate := range rates {
		s.processAndCacheRate(from, to, rate)
	}

	s.logger.Info("Fetched and cached rates from provider",
		"from", from, "count", len(rates))
	return rates, nil
}

func (s *Service) IsSupported(from, to string) bool {
	if from == to {
		return true
	}
	return s.provider.IsSupported(from, to)
}

// ---- Private Service Methods ----

func (s *Service) getValidCachedRate(
	entity registry.Entity,
	from, to string,
	ttl time.Duration,
) (*provider.ExchangeInfo, bool) {
	info, ok := entity.(*ExchangeRateInfo)
	if !ok {
		s.logger.Error(
			"Invalid cache entry type",
			"from", from,
			"to", to,
			"type", fmt.Sprintf("%T", entity))
		return nil, false
	}

	// Check if cached rate is expired
	if time.Now().After(info.Timestamp.Add(ttl)) {
		s.logger.Debug("Cached rate expired", "from", from, "to", to, "timestamp", info.Timestamp)
		return nil, false
	}

	s.logger.Info(
		"Exchange rate retrieved from cache",
		"from", from,
		"to", to,
		"rate", info.Rate,
	)

	// Convert to provider.ExchangeInfo
	return &provider.ExchangeInfo{
		OriginalCurrency:  from,
		ConvertedCurrency: to,
		ConversionRate:    info.Rate,
		Source:            info.Source,
		Timestamp:         info.Timestamp,
	}, true
}

func (s *Service) getRateFromCache(
	ctx context.Context,
	from, to string,
) (*provider.ExchangeInfo, bool) {
	log := s.logger.With(
		"to", to,
		"from", from,
		"provider", s.provider.Name(),
	)
	cacheKey := fmt.Sprintf("%s:%s", from, to)
	log.Debug("Checking registry for rate", "key", cacheKey)

	entity, err := s.exchangeRegistry.Get(ctx, cacheKey)
	if err != nil {
		log.Debug("Rate not found in registry (error)", "key", cacheKey, "error", err)
		return nil, false
	}

	if entity == nil {
		log.Debug("Rate not found in registry", "key", cacheKey)
		return nil, false
	}

	cached, ok := s.getValidCachedRate(entity, from, to, 24*time.Hour)
	if !ok {
		log.Debug("Cached rate is invalid or expired", "key", cacheKey)
		return nil, false
	}

	// Convert ExchangeRateInfo to provider.ExchangeInfo
	return &provider.ExchangeInfo{
		OriginalCurrency:  from,
		ConvertedCurrency: to,
		ConversionRate:    cached.ConversionRate,
		Source:            cached.Source,
		Timestamp:         cached.Timestamp,
	}, true
}

// validateFetchedRate checks if the fetched rate is valid.
func validateFetchedRate(rate *provider.ExchangeInfo) error {
	if rate.ConversionRate <= 0 ||
		math.IsNaN(rate.ConversionRate) ||
		math.IsInf(rate.ConversionRate, 0) {
		return fmt.Errorf("invalid rate received from provider: %f", rate.ConversionRate)
	}
	return nil
}

// saveDirectAndInverseRates is deprecated and will be removed in a future version.
// Please use exchangeCache.CacheRates instead.
// This method is kept for backward compatibility.
func (s *Service) saveDirectAndInverseRates(
	ctx context.Context,
	from, to string,
	rateInfo *ExchangeRateInfo,
) {
	s.logger.Warn("saveDirectAndInverseRates is deprecated, use exchangeCache.CacheRates instead",
		"from", from, "to", to)

	// Convert ExchangeRateInfo to provider.ExchangeInfo for the cache
	rate := &provider.ExchangeInfo{
		OriginalCurrency:  from,
		ConvertedCurrency: to,
		ConversionRate:    rateInfo.Rate,
		Source:            rateInfo.Source,
		Timestamp:         rateInfo.Timestamp,
	}

	// Use the exchange cache to handle the caching
	rates := map[string]*provider.ExchangeInfo{
		to: rate,
	}

	if err := s.exchangeCache.CacheRates(ctx, rates, rateInfo.Source); err != nil {
		s.logger.Error("Failed to save rates using exchange cache",
			"from", from,
			"to", to,
			"error", err)
	}
}
