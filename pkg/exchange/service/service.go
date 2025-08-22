package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/exchange/core"
	"github.com/amirasaad/fintech/pkg/provider/exchange"
)

// Service handles currency exchange operations using a provider and cache
type Service struct {
	provider exchange.Exchange // Single provider that may be a composite
	cache    *exchange.Cache   // Optional cache
	logger   *slog.Logger      // Logger for the service
}

// New creates a new exchange service with the given provider and cache
func New(
	provider exchange.Exchange,
	cache *exchange.Cache,
	logger *slog.Logger,
) *Service {
	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		provider: provider,
		cache:    cache,
		logger:   logger,
	}
}

// Convert converts an amount from one currency to another
func (s *Service) Convert(
	ctx context.Context,
	from, to string,
	amount float64,
) (*core.ConversionResult, error) {
	if amount <= 0 {
		return nil, core.ErrInvalidAmount
	}

	rate, err := s.GetRate(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange rate: %w", err)
	}

	return &core.ConversionResult{
		FromAmount: amount,
		ToAmount:   amount * rate.Value,
		Rate:       rate.Value,
		Source:     rate.Source,
	}, nil
}

// GetRate gets the exchange rate between two currencies
func (s *Service) GetRate(
	ctx context.Context,
	from, to string,
) (*core.Rate, error) {
	// Try cache first
	if s.cache != nil {
		if rate, err := s.cache.GetRate(ctx, from, to); err == nil {
			return &core.Rate{
				From:      from,
				To:        to,
				Value:     rate.Rate,
				Timestamp: rate.Timestamp,
				Source:    rate.Provider,
			}, nil
		}
	}

	// Get rate from provider
	rateInfo, err := s.provider.FetchRate(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("provider error: %w", err)
	}

	rate := &core.Rate{
		From:      from,
		To:        to,
		Value:     rateInfo.Rate,
		Timestamp: rateInfo.Timestamp,
		Source:    rateInfo.Provider,
	}

	// Update cache
	if s.cache != nil {
		if err := s.cache.StoreRate(ctx, rateInfo); err != nil {
			s.logger.Error("failed to cache rate", "error", err)
		}
	}

	return rate, nil
}

// GetRates gets multiple exchange rates in a single request
func (s *Service) GetRates(
	ctx context.Context,
	from string,
	to []string,
) (map[string]*core.Rate, error) {
	if len(to) == 0 {
		return nil, errors.New("no target currencies provided")
	}

	// Check cache first
	cachedRates := make(map[string]*core.Rate)
	var toFetch []string

	if s.cache != nil {
		rates, err := s.cache.BatchGetRates(ctx, from, to)
		if err == nil {
			for currency, rate := range rates {
				if rate != nil {
					cachedRates[currency] = &core.Rate{
						From:      from,
						To:        currency,
						Value:     rate.Rate,
						Timestamp: rate.Timestamp,
						Source:    rate.Provider,
					}
				} else {
					toFetch = append(toFetch, currency)
				}
			}
		} else {
			toFetch = to
		}
	} else {
		toFetch = to
	}

	// If we have everything cached, return early
	if len(cachedRates) == len(to) {
		return cachedRates, nil
	}

	// Fetch missing rates from provider
	fetchedRates := make(map[string]*core.Rate)
	if len(toFetch) > 0 {
		rates, err := s.provider.FetchRates(ctx, from, toFetch)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch rates: %w", err)
		}

		// Convert RateInfo to core.Rate and update cache
		for currency, rateInfo := range rates {
			fetchedRates[currency] = &core.Rate{
				From:      from,
				To:        currency,
				Value:     rateInfo.Rate,
				Timestamp: rateInfo.Timestamp,
				Source:    rateInfo.Provider,
			}

			// Update cache
			if s.cache != nil {
				if err := s.cache.StoreRate(ctx, rateInfo); err != nil {
					s.logger.Error("failed to cache rate", "error", err)
				}
			}
		}
	}

	// Merge cached and fetched rates
	result := make(map[string]*core.Rate, len(to))
	for _, currency := range to {
		if rate, exists := cachedRates[currency]; exists {
			result[currency] = rate
		} else if rate, exists := fetchedRates[currency]; exists {
			result[currency] = rate
		} else {
			result[currency] = nil // Indicate missing rate
		}
	}

	return result, nil
}
