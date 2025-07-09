package currency

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/amirasaad/fintech/pkg/registry"
)

// Common errors
var (
	ErrInvalidCurrencyCode = errors.New("invalid currency code: must be 3 uppercase letters")
	ErrInvalidDecimals     = errors.New("invalid decimals: must be between 0 and 8")
	ErrInvalidSymbol       = errors.New("invalid symbol: must not be empty and max 10 characters")
	ErrCurrencyNotFound    = errors.New("currency not found")
	ErrCurrencyExists      = errors.New("currency already exists")
)

const (
	// DefaultCurrency is the fallback currency code (USD)
	DefaultCurrency = "USD"
	// DefaultDecimals is the default number of decimal places for currencies
	DefaultDecimals = 2
	// MaxDecimals is the maximum number of decimal places allowed
	MaxDecimals = 8
	// MaxSymbolLength is the maximum length for currency symbols
	MaxSymbolLength = 10
)

// CurrencyCode represents a 3-letter ISO currency code
type CurrencyCode string

// CurrencyMeta holds currency-specific metadata
type CurrencyMeta struct {
	Code     string            `json:"code"`
	Name     string            `json:"name"`
	Symbol   string            `json:"symbol"`
	Decimals int               `json:"decimals"`
	Country  string            `json:"country,omitempty"`
	Region   string            `json:"region,omitempty"`
	Active   bool              `json:"active"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Created  time.Time         `json:"created"`
	Updated  time.Time         `json:"updated"`
}

// CurrencyEntity implements the registry.Entity interface
type CurrencyEntity struct {
	*registry.BaseEntity
	meta CurrencyMeta
}

// NewCurrencyEntity creates a new currency entity
func NewCurrencyEntity(meta CurrencyMeta) *CurrencyEntity {
	now := time.Now()
	meta.Created = now
	meta.Updated = now

	return &CurrencyEntity{
		BaseEntity: registry.NewBaseEntity(meta.Code, meta.Name),
		meta:       meta,
	}
}

// GetID returns the currency code
func (c *CurrencyEntity) GetID() string {
	return c.meta.Code
}

// GetName returns the currency name
func (c *CurrencyEntity) GetName() string {
	return c.meta.Name
}

// IsActive returns whether the currency is active
func (c *CurrencyEntity) IsActive() bool {
	return c.meta.Active
}

// GetMetadata returns currency metadata
func (c *CurrencyEntity) GetMetadata() map[string]string {
	metadata := c.BaseEntity.GetMetadata()
	metadata["code"] = c.meta.Code
	metadata["symbol"] = c.meta.Symbol
	metadata["decimals"] = strconv.Itoa(c.meta.Decimals)
	metadata["country"] = c.meta.Country
	metadata["region"] = c.meta.Region
	metadata["active"] = strconv.FormatBool(c.meta.Active)
	metadata["created"] = c.meta.Created.Format(time.RFC3339)
	metadata["updated"] = c.meta.Updated.Format(time.RFC3339)

	// Add custom metadata
	for k, v := range c.meta.Metadata {
		metadata[k] = v
	}

	return metadata
}

// GetCreatedAt returns the creation timestamp
func (c *CurrencyEntity) GetCreatedAt() time.Time {
	return c.meta.Created
}

// GetUpdatedAt returns the last update timestamp
func (c *CurrencyEntity) GetUpdatedAt() time.Time {
	return c.meta.Updated
}

// GetMeta returns the currency metadata
func (c *CurrencyEntity) GetMeta() CurrencyMeta {
	return c.meta
}

// CurrencyValidator implements registry.RegistryValidator for currency entities
type CurrencyValidator struct{}

// NewCurrencyValidator creates a new currency validator
func NewCurrencyValidator() *CurrencyValidator {
	return &CurrencyValidator{}
}

// Validate validates a currency entity
func (cv *CurrencyValidator) Validate(ctx context.Context, entity registry.Entity) error {
	currencyEntity, ok := entity.(*CurrencyEntity)
	if !ok {
		return fmt.Errorf("invalid entity type: expected *CurrencyEntity")
	}

	return validateCurrencyMeta(currencyEntity.GetMeta())
}

// ValidateMetadata validates currency metadata
func (cv *CurrencyValidator) ValidateMetadata(ctx context.Context, metadata map[string]string) error {
	// Validate required metadata fields
	requiredFields := []string{"code", "symbol", "decimals"}
	for _, field := range requiredFields {
		if value, exists := metadata[field]; !exists || value == "" {
			return fmt.Errorf("required metadata field missing: %s", field)
		}
	}

	// Validate currency code format
	if code, exists := metadata["code"]; exists {
		if !isValidCurrencyCode(code) {
			return ErrInvalidCurrencyCode
		}
	}

	// Validate decimals
	if decimalsStr, exists := metadata["decimals"]; exists {
		if decimals, err := strconv.Atoi(decimalsStr); err != nil {
			return ErrInvalidDecimals
		} else if decimals < 0 || decimals > MaxDecimals {
			return ErrInvalidDecimals
		}
	}

	// Validate symbol
	if symbol, exists := metadata["symbol"]; exists {
		if symbol == "" || len(symbol) > MaxSymbolLength {
			return ErrInvalidSymbol
		}
	}

	return nil
}

// validateCurrencyMeta validates currency metadata
func validateCurrencyMeta(meta CurrencyMeta) error {
	// Validate currency code format
	if !isValidCurrencyCode(meta.Code) {
		return ErrInvalidCurrencyCode
	}

	// Validate decimals
	if meta.Decimals < 0 || meta.Decimals > MaxDecimals {
		return ErrInvalidDecimals
	}

	// Validate symbol
	if meta.Symbol == "" || len(meta.Symbol) > MaxSymbolLength {
		return ErrInvalidSymbol
	}

	// Validate name
	if meta.Name == "" {
		return errors.New("currency name cannot be empty")
	}

	return nil
}

// IsValidCurrencyFormat returns true if the code is a well-formed ISO 4217 currency code (3 uppercase letters).
func IsValidCurrencyFormat(code string) bool {
	re := regexp.MustCompile(`^[A-Z]{3}$`)
	return re.MatchString(code)
}

// isValidCurrencyCode checks if a currency code is valid (3 uppercase letters)
func isValidCurrencyCode(code string) bool {
	return IsValidCurrencyFormat(code)
}

// CurrencyRegistry provides currency-specific operations using the registry system
type CurrencyRegistry struct {
	registry registry.RegistryProvider
	ctx      context.Context
}

// NewCurrencyRegistry creates a new currency registry with default currencies
func NewCurrencyRegistry(ctx context.Context) (*CurrencyRegistry, error) {
	// Create registry with currency-specific configuration
	config := registry.RegistryConfig{
		Name:             "currency-registry",
		MaxEntities:      1000,
		EnableEvents:     true,
		EnableValidation: true,
		CacheSize:        100,
		CacheTTL:         10 * time.Minute,
	}

	reg := registry.NewEnhancedRegistry(config)
	reg.WithValidator(NewCurrencyValidator())
	reg.WithCache(registry.NewMemoryCache(10 * time.Minute))

	cr := &CurrencyRegistry{
		registry: reg,
		ctx:      ctx,
	}

	// Register default currencies
	if err := cr.registerDefaults(); err != nil {
		return nil, fmt.Errorf("failed to register default currencies: %w", err)
	}

	return cr, nil
}

// NewCurrencyRegistryWithPersistence creates a currency registry with persistence
func NewCurrencyRegistryWithPersistence(ctx context.Context, persistencePath string) (*CurrencyRegistry, error) {
	config := registry.RegistryConfig{
		Name:              "currency-registry",
		MaxEntities:       1000,
		EnableEvents:      true,
		EnableValidation:  true,
		CacheSize:         100,
		CacheTTL:          10 * time.Minute,
		EnablePersistence: true,
		PersistencePath:   persistencePath,
		AutoSaveInterval:  time.Minute,
	}

	reg := registry.NewEnhancedRegistry(config)
	reg.WithValidator(NewCurrencyValidator())
	reg.WithCache(registry.NewMemoryCache(10 * time.Minute))

	// Add persistence
	persistence := registry.NewFilePersistence(persistencePath)
	reg.WithPersistence(persistence)

	cr := &CurrencyRegistry{
		registry: reg,
		ctx:      ctx,
	}

	// Load existing currencies from persistence
	entities, err := persistence.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load currencies from persistence: %w", err)
	}

	// Register loaded currencies
	for _, entity := range entities {
		if err := reg.Register(ctx, entity); err != nil {
			return nil, fmt.Errorf("failed to register loaded currency: %w", err)
		}
	}

	// If no currencies were loaded, register defaults
	if len(entities) == 0 {
		if err := cr.registerDefaults(); err != nil {
			return nil, fmt.Errorf("failed to register default currencies: %w", err)
		}
	}

	return cr, nil
}

// registerDefaults registers the default set of currencies
func (cr *CurrencyRegistry) registerDefaults() error {
	defaultCurrencies := []CurrencyMeta{
		{Code: "USD", Name: "US Dollar", Symbol: "$", Decimals: 2, Country: "United States", Region: "North America", Active: true},
		{Code: "EUR", Name: "Euro", Symbol: "€", Decimals: 2, Country: "European Union", Region: "Europe", Active: true},
		{Code: "GBP", Name: "British Pound", Symbol: "£", Decimals: 2, Country: "United Kingdom", Region: "Europe", Active: true},
		{Code: "JPY", Name: "Japanese Yen", Symbol: "¥", Decimals: 0, Country: "Japan", Region: "Asia", Active: true},
		{Code: "CAD", Name: "Canadian Dollar", Symbol: "C$", Decimals: 2, Country: "Canada", Region: "North America", Active: true},
		{Code: "AUD", Name: "Australian Dollar", Symbol: "A$", Decimals: 2, Country: "Australia", Region: "Oceania", Active: true},
		{Code: "CHF", Name: "Swiss Franc", Symbol: "CHF", Decimals: 2, Country: "Switzerland", Region: "Europe", Active: true},
		{Code: "CNY", Name: "Chinese Yuan", Symbol: "¥", Decimals: 2, Country: "China", Region: "Asia", Active: true},
		{Code: "INR", Name: "Indian Rupee", Symbol: "₹", Decimals: 2, Country: "India", Region: "Asia", Active: true},
		{Code: "BRL", Name: "Brazilian Real", Symbol: "R$", Decimals: 2, Country: "Brazil", Region: "South America", Active: true},
		{Code: "KWD", Name: "Kuwaiti Dinar", Symbol: "د.ك", Decimals: 3, Country: "Kuwait", Region: "Middle East", Active: true},
		{Code: "EGP", Name: "Egyptian Pound", Symbol: "£", Decimals: 2, Country: "Egypt", Region: "Africa", Active: true},
	}

	for _, meta := range defaultCurrencies {
		if err := cr.Register(meta); err != nil {
			return fmt.Errorf("failed to register %s: %w", meta.Code, err)
		}
	}

	return nil
}

// Register adds or updates a currency in the registry
func (cr *CurrencyRegistry) Register(meta CurrencyMeta) error {
	// Validate currency metadata
	if err := validateCurrencyMeta(meta); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Create currency entity
	entity := NewCurrencyEntity(meta)

	// Register with the registry
	if err := cr.registry.Register(cr.ctx, entity); err != nil {
		return fmt.Errorf("failed to register currency: %w", err)
	}

	return nil
}

// Get returns currency metadata for the given code
func (cr *CurrencyRegistry) Get(code string) (CurrencyMeta, error) {
	entity, err := cr.registry.Get(cr.ctx, code)
	if err != nil {
		return CurrencyMeta{}, fmt.Errorf("currency not found: %w", err)
	}

	// Convert entity back to currency metadata
	currencyEntity, ok := entity.(*CurrencyEntity)
	if !ok {
		// Fallback: try to convert from BaseEntity
		metadata := entity.GetMetadata()
		decimals, _ := strconv.Atoi(metadata["decimals"])
		active, _ := strconv.ParseBool(metadata["active"])

		return CurrencyMeta{
			Code:     metadata["code"],
			Name:     entity.GetName(),
			Symbol:   metadata["symbol"],
			Decimals: decimals,
			Country:  metadata["country"],
			Region:   metadata["region"],
			Active:   active,
		}, nil
	}

	return currencyEntity.GetMeta(), nil
}

// IsSupported checks if a currency code is registered and active
func (cr *CurrencyRegistry) IsSupported(code string) bool {
	if !cr.registry.IsRegistered(cr.ctx, code) {
		return false
	}

	entity, err := cr.registry.Get(cr.ctx, code)
	if err != nil {
		return false
	}

	return entity.IsActive()
}

// ListSupported returns a list of all supported currency codes
func (cr *CurrencyRegistry) ListSupported() ([]string, error) {
	entities, err := cr.registry.ListActive(cr.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list currencies: %w", err)
	}

	codes := make([]string, len(entities))
	for i, entity := range entities {
		codes[i] = entity.GetID()
	}

	return codes, nil
}

// ListAll returns all registered currencies (active and inactive)
func (cr *CurrencyRegistry) ListAll() ([]CurrencyMeta, error) {
	entities, err := cr.registry.List(cr.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list currencies: %w", err)
	}

	currencies := make([]CurrencyMeta, len(entities))
	for i, entity := range entities {
		if currencyEntity, ok := entity.(*CurrencyEntity); ok {
			currencies[i] = currencyEntity.GetMeta()
		} else {
			// Fallback conversion
			metadata := entity.GetMetadata()
			decimals, _ := strconv.Atoi(metadata["decimals"])
			active, _ := strconv.ParseBool(metadata["active"])

			currencies[i] = CurrencyMeta{
				Code:     metadata["code"],
				Name:     entity.GetName(),
				Symbol:   metadata["symbol"],
				Decimals: decimals,
				Country:  metadata["country"],
				Region:   metadata["region"],
				Active:   active,
			}
		}
	}

	return currencies, nil
}

// Unregister removes a currency from the registry
func (cr *CurrencyRegistry) Unregister(code string) error {
	if err := cr.registry.Unregister(cr.ctx, code); err != nil {
		return fmt.Errorf("failed to unregister currency: %w", err)
	}

	return nil
}

// Activate activates a currency
func (cr *CurrencyRegistry) Activate(code string) error {
	if err := cr.registry.Activate(cr.ctx, code); err != nil {
		return fmt.Errorf("failed to activate currency: %w", err)
	}

	return nil
}

// Deactivate deactivates a currency
func (cr *CurrencyRegistry) Deactivate(code string) error {
	if err := cr.registry.Deactivate(cr.ctx, code); err != nil {
		return fmt.Errorf("failed to deactivate currency: %w", err)
	}

	return nil
}

// Count returns the total number of registered currencies
func (cr *CurrencyRegistry) Count() (int, error) {
	return cr.registry.Count(cr.ctx)
}

// CountActive returns the number of active currencies
func (cr *CurrencyRegistry) CountActive() (int, error) {
	return cr.registry.CountActive(cr.ctx)
}

// Search searches for currencies by name
func (cr *CurrencyRegistry) Search(query string) ([]CurrencyMeta, error) {
	entities, err := cr.registry.Search(cr.ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search currencies: %w", err)
	}

	currencies := make([]CurrencyMeta, len(entities))
	for i, entity := range entities {
		if currencyEntity, ok := entity.(*CurrencyEntity); ok {
			currencies[i] = currencyEntity.GetMeta()
		}
	}

	return currencies, nil
}

// SearchByRegion searches for currencies by region
func (cr *CurrencyRegistry) SearchByRegion(region string) ([]CurrencyMeta, error) {
	entities, err := cr.registry.SearchByMetadata(cr.ctx, map[string]string{"region": region})
	if err != nil {
		return nil, fmt.Errorf("failed to search currencies by region: %w", err)
	}

	currencies := make([]CurrencyMeta, len(entities))
	for i, entity := range entities {
		if currencyEntity, ok := entity.(*CurrencyEntity); ok {
			currencies[i] = currencyEntity.GetMeta()
		}
	}

	return currencies, nil
}

// GetRegistry returns the underlying registry provider
func (cr *CurrencyRegistry) GetRegistry() registry.RegistryProvider {
	return cr.registry
}

// Global currency registry instance
var globalCurrencyRegistry *CurrencyRegistry

// Initialize global registry
func init() {
	var err error
	globalCurrencyRegistry, err = NewCurrencyRegistry(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to initialize global currency registry: %v", err))
	}
}

// Global convenience functions with error handling
func Register(meta CurrencyMeta) error {
	return globalCurrencyRegistry.Register(meta)
}

func Get(code string) (CurrencyMeta, error) {
	return globalCurrencyRegistry.Get(code)
}

func IsSupported(code string) bool {
	return globalCurrencyRegistry.IsSupported(code)
}

func ListSupported() ([]string, error) {
	return globalCurrencyRegistry.ListSupported()
}

func ListAll() ([]CurrencyMeta, error) {
	return globalCurrencyRegistry.ListAll()
}

func Unregister(code string) error {
	return globalCurrencyRegistry.Unregister(code)
}

func Count() (int, error) {
	return globalCurrencyRegistry.Count()
}

func CountActive() (int, error) {
	return globalCurrencyRegistry.CountActive()
}

func Search(query string) ([]CurrencyMeta, error) {
	return globalCurrencyRegistry.Search(query)
}

func SearchByRegion(region string) ([]CurrencyMeta, error) {
	return globalCurrencyRegistry.SearchByRegion(region)
}

// Backward compatibility functions (deprecated)
func RegisterLegacy(code string, meta CurrencyMeta) {
	// Convert legacy format to new format
	newMeta := CurrencyMeta{
		Code:     code,
		Name:     code,
		Symbol:   meta.Symbol,
		Decimals: meta.Decimals,
		Active:   true,
	}

	if err := Register(newMeta); err != nil {
		// Log error but don't panic for backward compatibility
		fmt.Printf("Warning: failed to register currency %s: %v\n", code, err)
	}
}

func GetLegacy(code string) CurrencyMeta {
	meta, err := Get(code)
	if err != nil {
		// Return default for backward compatibility
		return CurrencyMeta{
			Code:     code,
			Name:     code,
			Symbol:   code,
			Decimals: DefaultDecimals,
			Active:   false,
		}
	}
	return meta
}

func IsSupportedLegacy(code string) bool {
	return IsSupported(code)
}

func ListSupportedLegacy() []string {
	codes, err := ListSupported()
	if err != nil {
		return []string{}
	}
	return codes
}

func UnregisterLegacy(code string) bool {
	err := Unregister(code)
	return err == nil
}

func CountLegacy() int {
	count, err := Count()
	if err != nil {
		return 0
	}
	return count
}
