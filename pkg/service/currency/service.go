package currency

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/registry"
)

// ---- Entity ----

type Entity struct {
	registry.Entity
	Code     money.Code `json:"code"`
	Name     string     `json:"name"`
	Symbol   string     `json:"symbol"`
	Decimals int        `json:"decimals"`
	Country  string     `json:"country,omitempty"`
	Region   string     `json:"region,omitempty"`
	Active   bool       `json:"active"`
}

// Service provides business logic for currency operations
type Service struct {
	registry registry.Provider
	logger   *slog.Logger
}

// New creates a new currency service
func New(
	registry registry.Provider,
	logger *slog.Logger,
) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		registry: registry,
		logger:   logger.With("service", "Currency"),
	}
}

// Get retrieves currency information by code
func (s *Service) Get(ctx context.Context, code string) (*money.Currency, error) {
	entity, err := s.registry.Get(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to get currency: %w", err)
	}

	// Convert entity to currency.Meta
	meta, err := toCurrency(entity)
	if err != nil {
		return nil, fmt.Errorf("failed to convert entity: %w", err)
	}

	return meta, nil
}

// ListSupported returns all supported currency codes
func (s *Service) ListSupported(ctx context.Context) ([]string, error) {
	entities, err := s.registry.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list active currencies: %w", err)
	}

	codes := make([]string, 0, len(entities))
	for _, entity := range entities {
		codes = append(codes, entity.ID())
	}

	return codes, nil
}

// ListAll returns all registered currencies with full metadata
func (s *Service) ListAll(ctx context.Context) ([]*money.Currency, error) {
	entities, err := s.registry.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list currencies: %w", err)
	}

	metas := make([]*money.Currency, 0, len(entities))
	for _, entity := range entities {
		meta, err := toCurrency(entity)
		if err != nil {
			s.logger.Error("failed to convert entity to meta", "error", err, "id", entity.ID())
			continue
		}
		metas = append(metas, meta)
	}

	return metas, nil
}

// Register registers a new currency
func (s *Service) Register(ctx context.Context, meta Entity) error {
	// Create a new base entity
	entity := registry.NewBaseEntity(meta.Code.String(), meta.Name)

	// Set the active status on the entity
	entity.SetActive(meta.Active)

	// Set all metadata fields
	entity.SetMetadata("symbol", meta.Symbol)
	entity.SetMetadata("decimals", strconv.Itoa(meta.Decimals))
	entity.SetMetadata("country", meta.Country)
	entity.SetMetadata("region", meta.Region)
	entity.SetMetadata("active", strconv.FormatBool(meta.Active))

	// Store the entity in the registry
	return s.registry.Register(ctx, entity)
}

// Unregister removes a currency from the registry
func (s *Service) Unregister(ctx context.Context, code string) error {
	return s.registry.Unregister(ctx, code)
}

// Activate activates a currency
func (s *Service) Activate(ctx context.Context, code string) error {
	return s.registry.Activate(ctx, code)
}

// Deactivate deactivates a currency
func (s *Service) Deactivate(ctx context.Context, code string) error {
	return s.registry.Deactivate(ctx, code)
}

// IsSupported checks if a currency is both registered and active
func (s *Service) IsSupported(ctx context.Context, code string) bool {
	entity, err := s.registry.Get(ctx, code)
	if err != nil {
		return false
	}
	return entity.Active()
}

// Search searches for currencies by name
func (s *Service) Search(
	ctx context.Context,
	query string,
) ([]*money.Currency, error) {
	entities, err := s.registry.Search(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search currencies: %w", err)
	}

	metas := make([]*money.Currency, 0, len(entities))
	for _, entity := range entities {
		meta, err := toCurrency(entity)
		if err != nil {
			s.logger.Error(
				"failed to convert entity to meta",
				"error",
				err,
				"id",
				entity.ID(),
			)
			continue
		}
		metas = append(metas, meta)
	}

	return metas, nil
}

// SearchByRegion searches for currencies by region
func (s *Service) SearchByRegion(
	ctx context.Context,
	region string,
) ([]*money.Currency, error) {
	entities, err := s.registry.SearchByMetadata(ctx, map[string]string{"region": region})
	if err != nil {
		return nil, fmt.Errorf("failed to search currencies by region: %w", err)
	}

	metas := make([]*money.Currency, 0, len(entities))
	for _, entity := range entities {
		meta, err := toCurrency(entity)
		if err != nil {
			s.logger.Error(
				"failed to convert entity to meta",
				"error",
				err,
				"id",
				entity.ID(),
			)
			continue
		}
		metas = append(metas, meta)
	}

	return metas, nil
}

// GetStatistics returns currency statistics
func (s *Service) GetStatistics(
	ctx context.Context,
) (map[string]any, error) {
	total, err := s.registry.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count total currencies: %w", err)
	}

	active, err := s.registry.CountActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count active currencies: %w", err)
	}

	return map[string]any{
		"total_currencies":  total,
		"active_currencies": active,
	}, nil
}

// ValidateCode validates a currency code format
func (s *Service) ValidateCode(
	ctx context.Context,
	code string,
) error {
	if !money.Code(code).IsValid() {
		return money.ErrInvalidCurrency
	}
	return nil
}

// GetDefault returns the default currency information
func (s *Service) GetDefault(
	ctx context.Context,
) (*money.Currency, error) {
	entity, err := s.registry.Get(ctx, money.DefaultCode.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get default currency: %w", err)
	}
	return toCurrency(entity)
}

// toCurrency converts a registry.Entity to money.Currency
// Note: The Active field is not part of money.Currency, so we'll just return the currency info
// without the active status. The active status should be checked using IsSupported() instead.
func toCurrency(entity registry.Entity) (*money.Currency, error) {
	if entity == nil {
		return nil, fmt.Errorf("entity is nil")
	}

	// Get metadata
	metadata := entity.Metadata()

	// Parse decimals
	decimals := 2 // default
	if decStr, ok := metadata["decimals"]; ok {
		if d, err := strconv.Atoi(decStr); err == nil {
			decimals = d
		}
	}

	return &money.Currency{
		Code:     money.Code(entity.ID()),
		Decimals: decimals,
	}, nil
}
