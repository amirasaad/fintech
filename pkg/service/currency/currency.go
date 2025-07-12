package currency

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"
)

// CurrencyService provides business logic for currency operations
type CurrencyService struct {
	registry *currency.CurrencyRegistry
	logger   *slog.Logger
}

// NewCurrencyService creates a new currency service
func NewCurrencyService(registry *currency.CurrencyRegistry, logger *slog.Logger) *CurrencyService {
	return &CurrencyService{
		registry: registry,
		logger:   logger,
	}
}

// GetCurrency retrieves currency information by code
func (s *CurrencyService) GetCurrency(ctx context.Context, code string) (currency.CurrencyMeta, error) {
	return s.registry.Get(code)
}

// ListSupportedCurrencies returns all supported currency codes
func (s *CurrencyService) ListSupportedCurrencies(ctx context.Context) ([]string, error) {
	return s.registry.ListSupported()
}

// ListAllCurrencies returns all registered currencies with full metadata
func (s *CurrencyService) ListAllCurrencies(ctx context.Context) ([]currency.CurrencyMeta, error) {
	return s.registry.ListAll()
}

// RegisterCurrency registers a new currency
func (s *CurrencyService) RegisterCurrency(ctx context.Context, meta currency.CurrencyMeta) error {
	return s.registry.Register(meta)
}

// UnregisterCurrency removes a currency from the registry
func (s *CurrencyService) UnregisterCurrency(ctx context.Context, code string) error {
	return s.registry.Unregister(code)
}

// ActivateCurrency activates a currency
func (s *CurrencyService) ActivateCurrency(ctx context.Context, code string) error {
	return s.registry.Activate(code)
}

// DeactivateCurrency deactivates a currency
func (s *CurrencyService) DeactivateCurrency(ctx context.Context, code string) error {
	return s.registry.Deactivate(code)
}

// IsCurrencySupported checks if a currency is supported
func (s *CurrencyService) IsCurrencySupported(ctx context.Context, code string) bool {
	return s.registry.IsSupported(code)
}

// SearchCurrencies searches for currencies by name
func (s *CurrencyService) SearchCurrencies(ctx context.Context, query string) ([]currency.CurrencyMeta, error) {
	return s.registry.Search(query)
}

// SearchCurrenciesByRegion searches for currencies by region
func (s *CurrencyService) SearchCurrenciesByRegion(ctx context.Context, region string) ([]currency.CurrencyMeta, error) {
	return s.registry.SearchByRegion(region)
}

// GetCurrencyStatistics returns currency statistics
func (s *CurrencyService) GetCurrencyStatistics(ctx context.Context) (map[string]interface{}, error) {
	total, err := s.registry.Count()
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	active, err := s.registry.CountActive()
	if err != nil {
		return nil, fmt.Errorf("failed to get active count: %w", err)
	}

	return map[string]interface{}{
		"total_currencies":    total,
		"active_currencies":   active,
		"inactive_currencies": total - active,
	}, nil
}

// ValidateCurrencyCode validates a currency code format
func (s *CurrencyService) ValidateCurrencyCode(ctx context.Context, code string) error {
	if !currency.IsValidCurrencyFormat(code) {
		return currency.ErrInvalidCurrencyCode
	}
	return nil
}

// GetDefaultCurrency returns the default currency information
func (s *CurrencyService) GetDefaultCurrency(ctx context.Context) (currency.CurrencyMeta, error) {
	return s.registry.Get(currency.DefaultCurrency)
}

// GetRegistry returns the underlying currency registry
func (s *CurrencyService) GetRegistry() *currency.CurrencyRegistry {
	return s.registry
}
