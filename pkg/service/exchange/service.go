package exchange

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
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

// ---- Entity ----

type ExchangeRateInfo struct {
	registry.BaseEntity
	From      string
	To        string
	Rate      float64
	Source    string
	Timestamp time.Time
}

// NewExchangeRateInfo creates a new ExchangeRateInfo instance with the given parameters
func NewExchangeRateInfo(from, to string, rate float64, source string) *ExchangeRateInfo {
	id := fmt.Sprintf("%s:%s", from, to)
	return &ExchangeRateInfo{
		BaseEntity: *registry.NewBaseEntity(id, fmt.Sprintf("%s:%s", from, to)),
		From:       from,
		To:         to,
		Rate:       rate,
		Source:     source,
		Timestamp:  time.Now().UTC(),
	}
}

// ---- Conversion Helpers ----

func (e *ExchangeRateInfo) toProviderInfo() *provider.ExchangeInfo {
	return &provider.ExchangeInfo{
		OriginalCurrency:  e.From,
		ConvertedCurrency: e.To,
		ConversionRate:    e.Rate,
		Source:            e.Source,
		Timestamp:         e.Timestamp,
	}
}

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

func newExchangeRateInfo(from, to string, rate float64, source string) *ExchangeRateInfo {
	return &ExchangeRateInfo{
		From:      from,
		To:        to,
		Rate:      rate,
		Source:    source,
		Timestamp: time.Now(),
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
	exchangeRegistry registry.Provider // Specific registry for exchange rates
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
	}

	return s
}

// FetchAndCacheRates fetches and caches exchange rates for the given currency
func (s *Service) FetchAndCacheRates(
	ctx context.Context,
	from string,
) error {
	if err := s.validateProviderHealth(); err != nil {
		return err
	}
	s.logger.Info(
		"Fetching exchange rates from provider",
		"provider", s.provider.(interface{ Name() string }).Name(),
		"from", from,
	)
	rates, err := s.provider.GetRates(ctx, from, []string{})
	if err != nil {
		s.logger.Warn(
			"Failed to get rates from provider",
			"provider", s.provider.(interface{ Name() string }).Name(),
			"error", err,
		)
		return err
	}
	for to, rate := range rates {
		s.processAndCacheRate(from, to, rate)
	}
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
// It no longer caches the inverse rate as per the requirement.
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

	// Create and save the rate info
	rateInfo := newExchangeRateInfo(from, to, rate.ConversionRate, s.provider.Name())
	s.saveDirectAndInverseRates(context.Background(), from, to, rateInfo)

	s.logger.Info(
		"Successfully processed exchange rate",
		"provider", s.provider.Name(),
		"from", from,
		"to", to,
		"rate", rate.ConversionRate,
	)
}

// ---- Public Service Methods ----

func (s *Service) Name() string    { return "ExchangeService" }
func (s *Service) IsHealthy() bool { return true }

// Convert converts an amount from one currency to another.
// It first checks the cache for a valid rate, and if not found, fetches it from the provider.
func (s *Service) Convert(
	ctx context.Context,
	amount *money.Money,
	to currency.Code,
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
	if cached, ok := s.getRateFromCache(ctx, from, to); ok {
		return cached, nil
	}
	return s.fetchRateFromProviders(ctx, from, to)
}

// GetRates fetches multiple exchange rates in a single request.
// Implements provider.ExchangeRate interface.
func (s *Service) GetRates(
	ctx context.Context,
	from string,
	to []string,
) (map[string]*provider.ExchangeInfo, error) {
	result := make(map[string]*provider.ExchangeInfo, len(to))

	// If no target currencies provided, return empty result
	if len(to) == 0 {
		return result, nil
	}

	// Process each target currency
	for _, target := range to {
		rate, err := s.getRatesHelper(ctx, from, target)
		if err != nil {
			s.logger.Warn("Failed to get rate",
				"from", from,
				"to", target,
				"error", err)
			continue
		}
		result[target] = rate
	}

	return result, nil
}

// getRatesHelper contains the GetRates logic.
func (s *Service) getRatesHelper(
	ctx context.Context,
	from string,
	to string,
) (*provider.ExchangeInfo, error) {
	if from == "" || to == "" {
		return nil, errors.New(
			"from currency and to currency are required")
	}

	// Handle same currency case
	if from == to {
		return &provider.ExchangeInfo{
			OriginalCurrency:  from,
			ConvertedCurrency: to,
			ConversionRate:    1.0,
			Source:            "identity",
			Timestamp:         time.Now(),
		}, nil
	}

	// Try to get rate from cache first
	if cached, ok := s.getRateFromCache(ctx, from, to); ok {
		return cached, nil
	}

	// Fall back to provider
	info, err := s.provider.GetRate(ctx, from, to)
	if err != nil {
		s.logger.Warn("Failed to get rate from provider",
			"from", from,
			"to", to,
			"error", err)
		return nil, err
	}

	// Convert provider-specific rate info to our standard format
	rateInfo := &provider.ExchangeInfo{
		OriginalCurrency:  from,
		ConvertedCurrency: to,
		ConversionRate:    info.ConversionRate,
		Source:            info.Source,
		Timestamp:         time.Now(),
	}

	// Cache the rate
	s.saveDirectAndInverseRates(ctx, from, to, &ExchangeRateInfo{
		From:      from,
		To:        to,
		Rate:      info.ConversionRate,
		Source:    info.Source,
		Timestamp: time.Now(),
	})

	return rateInfo, nil
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
	if entity != nil {
		if cached, ok := s.getValidCachedRate(entity, from, to, 24*time.Hour); ok {
			// Convert ExchangeRateInfo to provider.ExchangeInfo
			return &provider.ExchangeInfo{
				OriginalCurrency:  from,
				ConvertedCurrency: to,
				ConversionRate:    cached.ConversionRate,
				Source:            cached.Source,
				Timestamp:         cached.Timestamp,
			}, true
		}
	}

	log.Debug("Rate not found in registry", "key", cacheKey)
	return nil, false
}

func (s *Service) fetchRateFromProviders(
	ctx context.Context,
	from, to string,
) (*provider.ExchangeInfo, error) {
	log := s.logger.With(
		"to", to,
		"from", from,
		"provider", s.provider.Name(),
	)

	if !s.provider.IsHealthy() {
		log.Warn(
			"Skipping unhealthy provider",
			"provider", s.provider.Name(),
		)
		return nil, fmt.Errorf("provider %s is unhealthy", s.provider.Name())
	}

	rate, err := s.provider.GetRate(ctx, from, to)
	if err != nil {
		log.Warn(
			"Failed to get rate from provider",
			"error", err,
		)
		return nil, fmt.Errorf("failed to get rate from provider: %w", err)
	}

	if err := validateFetchedRate(rate); err != nil {
		log.Warn(
			"Invalid rate from provider",
			"rate", rate.ConversionRate,
		)
		return nil, fmt.Errorf("invalid rate from provider: %w", err)
	}

	rateInfo := newExchangeRateInfo(from, to, rate.ConversionRate, s.provider.Name())
	log.Info(
		"Successfully fetched exchange rate",
		"from", from,
		"to", to,
		"rate", rate.ConversionRate,
	)

	// Save the direct rate to the registry
	s.saveDirectAndInverseRates(ctx, from, to, rateInfo)
	return rateInfo.toProviderInfo(), nil
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

// saveDirectAndInverseRates saves the direct rate to the registry.
// It no longer caches the inverse rate as per the requirement.
func (s *Service) saveDirectAndInverseRates(
	ctx context.Context,
	from, to string,
	rateInfo *ExchangeRateInfo,
) {
	log := s.logger.With("provider", s.provider.Name())
	cacheKey := fmt.Sprintf("%s:%s", from, to)
	log.Debug("Saving rate to registry", "key", cacheKey, "rate", rateInfo.Rate)

	if err := s.exchangeRegistry.Register(ctx, rateInfo); err != nil {
		log.Error("Failed to save rate to registry",
			"key", cacheKey,
			"from", from,
			"to", to,
			"error", err)
		return
	}

	log.Debug(
		"Successfully saved rate to registry",
		"key", cacheKey,
		"rate", rateInfo.Rate,
	)
}
