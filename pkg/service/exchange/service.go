package exchange

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/provider/exchange"
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

// Service handles currency exchange operations with cache-first approach
type Service struct {
	provider exchange.Exchange
	registry registry.Provider // Registry for cached exchange rates
	logger   *slog.Logger
}

// New creates a new exchange service with the given registry and provider
func New(
	registry registry.Provider,
	provider exchange.Exchange,
	log *slog.Logger,
) *Service {
	if log == nil {
		log = slog.Default()
	}

	return &Service{
		provider: provider,
		logger:   log,
		registry: registry,
	}
}

// processAndCacheRate validates, logs, and caches a rate with TTL support.
// It uses the exchange cache to handle the actual caching.
// This is a convenience method that wraps the bulk caching functionality
// for a single rate.
// The context is used for cancellation and deadline propagation to the registry.
func (s *Service) processAndCacheRate(
	ctx context.Context,
	from, to string,
	rate *exchange.RateInfo,
) {
	if rate == nil {
		err := fmt.Errorf("provider %s returned nil rate", s.provider.Metadata().Name)
		s.logger.Error("Failed to fetch exchange rate",
			"from", from,
			"to", to,
			"provider", s.provider.Metadata().Name,
			"error", err,
		)
		return
	}

	// Create rate info with current timestamp
	rateInfo := newExchangeRateInfo(from, to, rate.Rate, s.provider.Metadata().Name)

	// Store last updated timestamp in metadata
	rateInfo.SetMetadata("last_updated", time.Now().UTC().Format(time.RFC3339Nano))

	// Register the direct rate (from -> to)
	if err := s.registry.Register(
		ctx,
		rateInfo,
	); err != nil {
		s.logger.Error("Failed to cache exchange rate",
			"from", from,
			"to", to,
			"error", err,
		)
	}

	// Also register the inverse rate (to -> from) if not 1:1
	if math.Abs(rate.Rate) > 1e-10 { // Avoid division by zero
		inverseRate := 1.0 / rate.Rate
		// Create inverse rate info with current timestamp
		inverseInfo := newExchangeRateInfo(to, from, inverseRate, s.provider.Metadata().Name)
		inverseInfo.SetMetadata("last_updated", time.Now().UTC().Format(time.RFC3339Nano))

		if err := s.registry.Register(
			ctx,
			inverseInfo,
		); err != nil {
			s.logger.Error("Failed to cache inverse exchange rate",
				"from", to,
				"to", from,
				"error", err,
			)
		}
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
) (*money.Money, *exchange.RateInfo, error) {
	if err := validateAmount(amount); err != nil {
		return nil, nil, fmt.Errorf("invalid amount: %w", err)
	}

	from := amount.Currency().String()
	toStr := to.String()

	// Check if conversion is needed
	if from == toStr {
		return amount, &exchange.RateInfo{
			FromCurrency: from,
			ToCurrency:   toStr,
			Rate:         1.0,
			Provider:     "identity",
			Timestamp:    time.Now(),
		}, nil
	}

	// Try to get rate from cache first
	rate, err := s.GetRate(ctx, from, toStr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get exchange rate: %w", err)
	}

	// Convert the amount
	converted, err := amount.Multiply(rate.Rate)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert amount: %w", err)
	}

	result, err := money.New(converted.AmountFloat(), to)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create money: %w", err)
	}

	return result, rate, nil
}

// GetRate gets the exchange rate between two currencies with cache-first approach
func (s *Service) GetRate(
	ctx context.Context,
	from,
	to string,
) (*exchange.RateInfo, error) {
	// Check for invalid input
	if from == "" || to == "" {
		return nil, fmt.Errorf("invalid currency codes: from='%s', to='%s'", from, to)
	}

	// Check if it's the same currency
	if from == to {
		return &exchange.RateInfo{
			FromCurrency: from,
			ToCurrency:   to,
			Rate:         1.0,
			Provider:     "identity",
			Timestamp:    time.Now(),
		}, nil
	}

	// Try to get from cache first
	if rate, ok := s.getRateFromCache(ctx, from, to); ok {
		return rate, nil
	}

	// Fallback to provider
	if s.provider == nil {
		return nil, ErrNoProvidersAvailable
	}

	rate, err := s.provider.FetchRate(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rates from provider: %w", err)
	}

	s.processAndCacheRate(ctx, from, to, rate)
	return rate, nil
}

func (s *Service) IsSupported(from, to string) bool {
	if from == to {
		return true
	}
	return s.provider.IsSupported(from, to)
}

// ---- Private Service Methods ----

func (s *Service) getRateFromCache(
	ctx context.Context,
	from, to string,
) (*exchange.RateInfo, bool) {
	key := fmt.Sprintf("%s:%s", from, to)

	entity, err := s.registry.Get(ctx, key)
	if err != nil {
		s.logger.Debug("Cache miss (error)", "key", key, "error", err)
		return nil, false
	}

	if entity == nil {
		s.logger.Debug("Cache miss (not found)", "key", key)
		return nil, false
	}

	// Check if we can get the rate directly
	if rateInfo, ok := entity.(*ExchangeRateInfo); ok {
		s.logger.Debug("Cache hit", "key", key, "rate", rateInfo.Rate)
		return &exchange.RateInfo{
			FromCurrency: rateInfo.From,
			ToCurrency:   rateInfo.To,
			Rate:         rateInfo.Rate,
			Provider:     rateInfo.Source,
			Timestamp:    rateInfo.Timestamp,
		}, true
	}

	return nil, false
}
