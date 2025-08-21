package provider

import (
	"log/slog"

	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/amirasaad/fintech/pkg/registry"
)

// Deprecated: Use provider.ExchangeRate interface directly.
type ExchangeRateCurrencyConverter = provider.ExchangeRate

// exchangeRateService provides real-time exchange rates with caching and fallback providers.
//
// Deprecated: Use exchange.Service from github.com/amirasaad/fintech/pkg/service/exchange instead.
type exchangeRateService struct {
	providers []provider.ExchangeRate
	cache     registry.Provider
	logger    *slog.Logger
	cfg       *config.ExchangeRateProviders
}

// NewExchangeRateService creates a new exchange rate service.
//
// Deprecated: Use exchange.New from github.com/amirasaad/fintech/pkg/service/exchange instead.
func NewExchangeRateService(
	providers []provider.ExchangeRate,
	cache registry.Provider,
	logger *slog.Logger,
	cfg *config.ExchangeRateProviders,
) *exchangeRateService {
	return &exchangeRateService{
		providers: providers,
		cache:     cache,
		logger:    logger,
		cfg:       cfg,
	}
}

// GetRate retrieves an exchange rate, trying cache first, then providers in order.
//
// Deprecated: Use exchange.Service.GetRate instead.
func (s *exchangeRateService) GetRate(from, to string) (*domain.ConversionInfo, error) {
	// Implementation moved to exchange/service.go
	return nil, domain.ErrExchangeRateUnavailable
}

// GetRates retrieves multiple exchange rates efficiently.
//
// Deprecated: Use exchange.Service.GetRates instead.
func (s *exchangeRateService) GetRates(
	from string,
	to []string,
) (map[string]*domain.ConversionInfo, error) {
	// Implementation moved to exchange/service.go
	return nil, domain.ErrExchangeRateUnavailable
}

// Deprecated: Use NewExchangeRateAPIProvider instead.
func NewExchangeRateCurrencyConverter(
	exchangeRateService *exchangeRateService,
	fallback ExchangeRateCurrencyConverter,
	logger *slog.Logger,
) ExchangeRateCurrencyConverter {
	return NewExchangeRateAPIProvider(&config.ExchangeRateApi{}, logger)
}
