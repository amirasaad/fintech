package currency

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"
)

// Service provides business logic for currency operations
type Service struct {
	registry *currency.Registry
	logger   *slog.Logger
}

// New creates a new currency service
func New(
	registry *currency.Registry,
	logger *slog.Logger,
) *Service {
	return &Service{
		registry: registry,
		logger:   logger,
	}
}

// GetCurrency retrieves currency information by code
func (s *Service) GetCurrency(ctx context.Context, code string) (currency.Meta, error) {
	return s.registry.Get(code)
}

// ListSupportedCurrencies returns all supported currency codes
func (s *Service) ListSupportedCurrencies(ctx context.Context) ([]string, error) {
	return s.registry.ListSupported()
}

// ListAllCurrencies returns all registered currencies with full metadata
func (s *Service) ListAllCurrencies(ctx context.Context) ([]currency.Meta, error) {
	return s.registry.ListAll()
}

// RegisterCurrency registers a new currency
func (s *Service) RegisterCurrency(ctx context.Context, meta currency.Meta) error {
	return s.registry.Register(meta)
}

// UnregisterCurrency removes a currency from the registry
func (s *Service) UnregisterCurrency(ctx context.Context, code string) error {
	return s.registry.Unregister(code)
}

// ActivateCurrency activates a currency
func (s *Service) ActivateCurrency(ctx context.Context, code string) error {
	return s.registry.Activate(code)
}

// DeactivateCurrency deactivates a currency
func (s *Service) DeactivateCurrency(ctx context.Context, code string) error {
	return s.registry.Deactivate(code)
}

// IsCurrencySupported checks if a currency is supported
func (s *Service) IsCurrencySupported(ctx context.Context, code string) bool {
	return s.registry.IsSupported(code)
}

// SearchCurrencies searches for currencies by name
func (s *Service) SearchCurrencies(ctx context.Context, query string) ([]currency.Meta, error) {
	return s.registry.Search(query)
}

// SearchCurrenciesByRegion searches for currencies by region
func (s *Service) SearchCurrenciesByRegion(
	ctx context.Context,
	region string,
) ([]currency.Meta, error) {
	return s.registry.SearchByRegion(region)
}

// GetCurrencyStatistics returns currency statistics
func (s *Service) GetCurrencyStatistics(
	ctx context.Context,
) (map[string]any, error) {
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
func (s *Service) ValidateCurrencyCode(
	ctx context.Context,
	code string,
) error {
	if !currency.IsValidFormat(code) {
		return currency.ErrInvalidCode
	}
	return nil
}

// GetDefaultCurrency returns the default currency information
func (s *Service) GetDefaultCurrency(
	ctx context.Context,
) (currency.Meta, error) {
	return s.registry.Get(currency.DefaultCode)
}

// GetRegistry returns the underlying currency registry
func (s *Service) GetRegistry() *currency.Registry {
	return s.registry
}
