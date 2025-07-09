package registry

import (
	"context"
	"fmt"
	"time"
)

// CurrencyEntity represents a currency in the registry
type CurrencyEntity struct {
	*BaseEntity
	code     string
	symbol   string
	decimals int
	active   bool
}

// NewCurrencyEntity creates a new currency entity
func NewCurrencyEntity(code, name, symbol string, decimals int) *CurrencyEntity {
	return &CurrencyEntity{
		BaseEntity: NewBaseEntity(code, name),
		code:       code,
		symbol:     symbol,
		decimals:   decimals,
		active:     true,
	}
}

// GetMetadata returns currency metadata
func (c *CurrencyEntity) Metadata() map[string]string {
	metadata := c.BaseEntity.Metadata()
	metadata["code"] = c.code
	metadata["symbol"] = c.symbol
	metadata["decimals"] = fmt.Sprintf("%d", c.decimals)
	metadata["active"] = fmt.Sprintf("%t", c.active)
	return metadata
}

// IsActive returns whether the currency is active
func (c *CurrencyEntity) Active() bool {
	return c.active
}

// Refactor CurrencyEntity to use property-style getter methods and forward to BaseEntity.
// Already implemented above or not needed due to direct field access

// Remove property-style getter methods that conflict.

// CurrencyRegistryExample demonstrates using the registry for currency management
func CurrencyRegistryExample() {
	fmt.Println("=== Currency Registry Example ===")

	// Create a registry optimized for currency management
	config := RegistryConfig{
		Name:             "currency-registry",
		MaxEntities:      1000,
		EnableEvents:     true,
		EnableValidation: true,
		CacheSize:        100,
		CacheTTL:         10 * time.Minute,
	}

	registry := NewEnhancedRegistry(config)
	registry.WithValidator(NewCurrencyValidator())
	registry.WithCache(NewMemoryCache(10 * time.Minute))

	ctx := context.Background()

	// Register major currencies
	currencies := []Entity{
		NewCurrencyEntity("USD", "US Dollar", "$", 2),
		NewCurrencyEntity("EUR", "Euro", "€", 2),
		NewCurrencyEntity("GBP", "British Pound", "£", 2),
		NewCurrencyEntity("JPY", "Japanese Yen", "¥", 0),
		NewCurrencyEntity("CAD", "Canadian Dollar", "C$", 2),
		NewCurrencyEntity("AUD", "Australian Dollar", "A$", 2),
		NewCurrencyEntity("CHF", "Swiss Franc", "CHF", 2),
		NewCurrencyEntity("CNY", "Chinese Yuan", "¥", 2),
	}

	for _, currency := range currencies {
		err := registry.Register(ctx, currency)
		if err != nil {
			fmt.Printf("Failed to register %s: %v\n", currency.Name(), err)
		} else {
			fmt.Printf("Registered: %s (%s)\n", currency.Name(), currency.Metadata()["code"])
		}
	}

	// Demonstrate search capabilities
	fmt.Println("\n--- Search Examples ---")

	// Search by name
	dollarResults, _ := registry.Search(ctx, "Dollar")
	fmt.Printf("Found %d currencies with 'Dollar' in name:\n", len(dollarResults))
	for _, currency := range dollarResults {
		fmt.Printf("  - %s (%s)\n", currency.Name(), currency.Metadata()["code"])
	}

	// Search by metadata (2 decimal places)
	twoDecimalCurrencies, _ := registry.SearchByMetadata(ctx, map[string]string{"decimals": "2"})
	fmt.Printf("\nFound %d currencies with 2 decimal places:\n", len(twoDecimalCurrencies))
	for _, currency := range twoDecimalCurrencies {
		fmt.Printf("  - %s (%s)\n", currency.Name(), currency.Metadata()["code"])
	}

	// List all active currencies
	activeCurrencies, _ := registry.ListActive(ctx)
	fmt.Printf("\nFound %d active currencies:\n", len(activeCurrencies))
	for _, currency := range activeCurrencies {
		if ce, ok := currency.(*CurrencyEntity); ok {
			fmt.Printf("  - %s (%s) %s\n",
				ce.Name(),
				ce.ID(),
				ce.symbol)
		} else {
			fmt.Printf("  - %s (%s) %s\n",
				currency.Name(),
				currency.ID(),
				currency.Metadata()["symbol"])
		}
	}

	// Demonstrate metadata operations
	fmt.Println("\n--- Metadata Operations ---")

	// Add additional metadata to USD
	err := registry.SetMetadata(ctx, "USD", "country", "United States")
	if err != nil {
		fmt.Printf("Failed to set metadata: %v\n", err)
	}

	// Get the metadata
	country, err := registry.GetMetadata(ctx, "USD", "country")
	if err != nil {
		fmt.Printf("Failed to get metadata: %v\n", err)
	} else {
		fmt.Printf("USD country: %s\n", country)
	}

	// Demonstrate lifecycle operations
	fmt.Println("\n--- Lifecycle Operations ---")

	// Deactivate a currency
	err = registry.Deactivate(ctx, "JPY")
	if err != nil {
		fmt.Printf("Failed to deactivate JPY: %v\n", err)
	} else {
		fmt.Println("JPY deactivated")
	}

	// Check active count
	activeCount, _ := registry.CountActive(ctx)
	fmt.Printf("Active currencies: %d\n", activeCount)

	// Reactivate JPY
	err = registry.Activate(ctx, "JPY")
	if err != nil {
		fmt.Printf("Failed to activate JPY: %v\n", err)
	} else {
		fmt.Println("JPY activated")
	}

	// Demonstrate performance
	fmt.Println("\n--- Performance Test ---")

	// Register many currencies for performance testing
	start := time.Now()
	for i := 1; i <= 100; i++ {
		currency := NewCurrencyEntity(
			fmt.Sprintf("TEST%d", i),
			fmt.Sprintf("Test Currency %d", i),
			fmt.Sprintf("T%d", i),
			2,
		)
		registry.Register(ctx, currency)
	}
	registerTime := time.Since(start)
	fmt.Printf("Registered 100 currencies in %v\n", registerTime)

	// Test lookup performance
	start = time.Now()
	for i := 1; i <= 1000; i++ {
		registry.Get(ctx, "USD")
	}
	lookupTime := time.Since(start)
	fmt.Printf("1000 USD lookups in %v (avg: %v per lookup)\n",
		lookupTime, lookupTime/1000)

	// Demonstrate statistics
	fmt.Println("\n--- Registry Statistics ---")

	totalCount, _ := registry.Count(ctx)
	fmt.Printf("Total currencies: %d\n", totalCount)

	activeCount, _ = registry.CountActive(ctx)
	fmt.Printf("Active currencies: %d\n", activeCount)

	// List all currencies
	allCurrencies, _ := registry.List(ctx)
	fmt.Printf("\nAll registered currencies (%d):\n", len(allCurrencies))
	for _, currency := range allCurrencies {
		if ce, ok := currency.(*CurrencyEntity); ok {
			metadata := ce.Metadata()
			status := "Active"
			if !ce.Active() {
				status = "Inactive"
			}
			fmt.Printf("  %s: %s %s (%s decimals) - %s\n",
				ce.ID(),
				ce.symbol,
				ce.Name(),
				metadata["decimals"],
				status)
		} else {
			metadata := currency.Metadata()
			status := "Active"
			if !currency.Active() {
				status = "Inactive"
			}
			fmt.Printf("  %s: %s %s (%s decimals) - %s\n",
				currency.ID(),
				metadata["symbol"],
				currency.Name(),
				metadata["decimals"],
				status)
		}
	}
}

// CurrencyValidator implements custom validation for currencies
type CurrencyValidator struct {
	*SimpleValidator
}

// NewCurrencyValidator creates a new currency validator
func NewCurrencyValidator() *CurrencyValidator {
	validator := &CurrencyValidator{
		SimpleValidator: NewSimpleValidator(),
	}

	// Set required metadata for currencies
	validator.WithRequiredMetadata([]string{"code", "symbol", "decimals"})

	// Add custom validators
	validator.WithValidator("code", validateCurrencyCode)
	validator.WithValidator("symbol", validateCurrencySymbol)
	validator.WithValidator("decimals", validateDecimals)

	return validator
}

// Validate validates a currency entity
func (v *CurrencyValidator) Validate(ctx context.Context, entity Entity) error {
	// First run basic validation
	if err := v.SimpleValidator.Validate(ctx, entity); err != nil {
		return err
	}

	// Additional currency-specific validation
	metadata := entity.Metadata()

	// Check code format (3 uppercase letters)
	code := metadata["code"]
	if len(code) != 3 {
		return fmt.Errorf("currency code must be exactly 3 characters")
	}

	// Check decimals range
	decimals := metadata["decimals"]
	if decimals == "0" || decimals == "2" {
		// Valid decimal places
	} else {
		return fmt.Errorf("currency decimals must be 0 or 2")
	}

	return nil
}

// Validation functions
func validateCurrencyCode(code string) error {
	if len(code) != 3 {
		return fmt.Errorf("currency code must be exactly 3 characters")
	}

	for _, char := range code {
		if char < 'A' || char > 'Z' {
			return fmt.Errorf("currency code must contain only uppercase letters")
		}
	}

	return nil
}

func validateCurrencySymbol(symbol string) error {
	if len(symbol) == 0 {
		return fmt.Errorf("currency symbol cannot be empty")
	}
	if len(symbol) > 5 {
		return fmt.Errorf("currency symbol too long (max 5 characters)")
	}
	return nil
}

func validateDecimals(decimals string) error {
	if decimals != "0" && decimals != "2" {
		return fmt.Errorf("currency decimals must be 0 or 2")
	}
	return nil
}

// CurrencyRegistryWithPersistence demonstrates persistent currency storage
func CurrencyRegistryWithPersistence() {
	fmt.Println("\n=== Currency Registry with Persistence ===")

	// Create registry with file persistence
	config := RegistryConfig{
		Name:              "persistent-currency-registry",
		EnableEvents:      true,
		EnableValidation:  true,
		CacheSize:         50,
		CacheTTL:          5 * time.Minute,
		EnablePersistence: true,
		PersistencePath:   "/tmp/currencies.json",
		AutoSaveInterval:  time.Minute,
	}

	registry := NewEnhancedRegistry(config)
	registry.WithValidator(NewCurrencyValidator())
	registry.WithCache(NewMemoryCache(5 * time.Minute))

	ctx := context.Background()

	// Register some currencies
	currencies := []Entity{
		NewCurrencyEntity("USD", "US Dollar", "$", 2),
		NewCurrencyEntity("EUR", "Euro", "€", 2),
		NewCurrencyEntity("GBP", "British Pound", "£", 2),
	}

	for _, currency := range currencies {
		registry.Register(ctx, currency)
	}

	fmt.Printf("Registered %d currencies to persistent storage\n", len(currencies))

	// Simulate application restart by creating a new registry
	fmt.Println("Simulating application restart...")

	newRegistry := NewEnhancedRegistry(config)
	newRegistry.WithValidator(NewCurrencyValidator())
	newRegistry.WithCache(NewMemoryCache(5 * time.Minute))

	// Load currencies from persistence
	persistence := NewFilePersistence("/tmp/currencies.json")
	entities, err := persistence.Load(ctx)
	if err != nil {
		fmt.Printf("Failed to load currencies: %v\n", err)
		return
	}

	for _, entity := range entities {
		newRegistry.Register(ctx, entity)
	}

	fmt.Printf("Loaded %d currencies from persistent storage\n", len(entities))

	// Verify the currencies are available
	loadedCurrencies, _ := newRegistry.List(ctx)
	fmt.Println("Available currencies after restart:")
	for _, currency := range loadedCurrencies {
		if ce, ok := currency.(*CurrencyEntity); ok {
			fmt.Printf("  %s: %s %s\n",
				ce.ID(),
				ce.symbol,
				ce.Name())
		} else {
			metadata := currency.Metadata()
			fmt.Printf("  %s: %s %s\n",
				currency.ID(),
				metadata["symbol"],
				currency.Name())
		}
	}
}

// CurrencyRegistryWithEvents demonstrates event-driven currency management
func CurrencyRegistryWithEvents() {
	fmt.Println("\n=== Currency Registry with Events ===")

	// Create registry with events
	config := RegistryConfig{
		Name:             "event-driven-currency-registry",
		EnableEvents:     true,
		EnableValidation: true,
		CacheSize:        50,
		CacheTTL:         5 * time.Minute,
	}

	registry := NewEnhancedRegistry(config)
	registry.WithValidator(NewCurrencyValidator())

	// Create event observer
	observer := &CurrencyEventLogger{}

	// In a real implementation, you'd subscribe to the event bus
	// For this example, we'll simulate events manually

	ctx := context.Background()

	// Register currencies and simulate events
	currencies := []Entity{
		NewCurrencyEntity("USD", "US Dollar", "$", 2),
		NewCurrencyEntity("EUR", "Euro", "€", 2),
	}

	for _, currency := range currencies {
		registry.Register(ctx, currency)
		observer.OnEntityRegistered(ctx, currency)
	}

	// Simulate currency updates
	usd, _ := registry.Get(ctx, "USD")
	if usd != nil {
		registry.SetMetadata(ctx, "USD", "last_updated", time.Now().Format(time.RFC3339))
		observer.OnEntityUpdated(ctx, usd)
	}

	// Simulate currency deactivation
	registry.Deactivate(ctx, "EUR")
	observer.OnEntityDeactivated(ctx, "EUR")

	fmt.Println("Currency events processed successfully")
}

// CurrencyEventLogger implements RegistryObserver for currency events
type CurrencyEventLogger struct{}

func (l *CurrencyEventLogger) OnEntityRegistered(ctx context.Context, entity Entity) {
	if ce, ok := entity.(*CurrencyEntity); ok {
		fmt.Printf("CURRENCY REGISTERED: %s (%s) %s\n",
			ce.Name(),
			ce.ID(),
			ce.symbol)
	} else {
		metadata := entity.Metadata()
		fmt.Printf("CURRENCY REGISTERED: %s (%s) %s\n",
			entity.Name(),
			entity.ID(),
			metadata["symbol"])
	}
}

func (l *CurrencyEventLogger) OnEntityUnregistered(ctx context.Context, id string) {
	fmt.Printf("CURRENCY UNREGISTERED: %s\n", id)
}

func (l *CurrencyEventLogger) OnEntityUpdated(ctx context.Context, entity Entity) {
	if ce, ok := entity.(*CurrencyEntity); ok {
		fmt.Printf("CURRENCY UPDATED: %s (%s)\n",
			ce.Name(),
			ce.ID())
	} else {
		fmt.Printf("CURRENCY UPDATED: %s (%s)\n",
			entity.Name(),
			entity.ID())
	}
}

func (l *CurrencyEventLogger) OnEntityActivated(ctx context.Context, id string) {
	fmt.Printf("CURRENCY ACTIVATED: %s\n", id)
}

func (l *CurrencyEventLogger) OnEntityDeactivated(ctx context.Context, id string) {
	fmt.Printf("CURRENCY DEACTIVATED: %s\n", id)
}
