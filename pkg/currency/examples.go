package currency

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ExampleBasicUsage demonstrates basic currency operations
func ExampleBasicUsage() {
	// Get currency information
	usd, err := Get("USD")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("USD: %s (%s) - %d decimals\n", usd.Name, usd.Symbol, usd.Decimals)

	// Check if currency is supported
	if IsSupported("EUR") {
		fmt.Println("EUR is supported")
	}

	// List all supported currencies
	supported, err := ListSupported()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Supported currencies: %v\n", supported)
}

// ExampleCustomRegistry demonstrates creating a custom currency registry
func ExampleCustomRegistry() {
	ctx := context.Background()

	// Create a new registry
	registry, err := NewCurrencyRegistry(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Register a custom cryptocurrency
	crypto := CurrencyMeta{
		Code:     "BTC",
		Name:     "Bitcoin",
		Symbol:   "₿",
		Decimals: 8,
		Country:  "Global",
		Region:   "Digital",
		Active:   true,
		Metadata: map[string]string{
			"type":       "cryptocurrency",
			"blockchain": "Bitcoin",
			"max_supply": "21000000",
		},
	}

	if err := registry.Register(crypto); err != nil {
		log.Fatal(err)
	}

	// Retrieve the currency
	retrieved, err := registry.Get("BTC")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Registered: %s (%s) - %s\n",
		retrieved.Name, retrieved.Symbol, retrieved.Metadata["type"])
}

// ExamplePersistence demonstrates using persistence with currency registry
func ExamplePersistence() {
	ctx := context.Background()

	// Create registry with persistence
	registry, err := NewCurrencyRegistryWithPersistence(ctx, "./currencies.json")
	if err != nil {
		log.Fatal(err)
	}

	// Register a new currency
	newCurrency := CurrencyMeta{
		Code:     "CUSTOM",
		Name:     "Custom Currency",
		Symbol:   "C",
		Decimals: 2,
		Country:  "Custom Country",
		Region:   "Custom Region",
		Active:   true,
	}

	if err := registry.Register(newCurrency); err != nil {
		log.Fatal(err)
	}

	// The currency will be automatically saved to the file
	fmt.Println("Currency registered and persisted")

	// Later, when you restart the application, the currency will be loaded
	// from the persistence file automatically
}

// ExampleSearch demonstrates searching for currencies
func ExampleSearch() {
	ctx := context.Background()
	registry, err := NewCurrencyRegistry(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Search by name
	results, err := registry.Search("Dollar")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Currencies with 'Dollar' in name:")
	for _, currency := range results {
		fmt.Printf("- %s (%s) from %s\n",
			currency.Name, currency.Symbol, currency.Country)
	}

	// Search by region
	european, err := registry.SearchByRegion("Europe")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\nEuropean currencies:")
	for _, currency := range european {
		fmt.Printf("- %s (%s)\n", currency.Name, currency.Symbol)
	}
}

// ExampleLifecycle demonstrates currency lifecycle management
func ExampleLifecycle() {
	ctx := context.Background()
	registry, err := NewCurrencyRegistry(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Register a currency as inactive
	currency := CurrencyMeta{
		Code:     "TEST",
		Name:     "Test Currency",
		Symbol:   "T",
		Decimals: 2,
		Active:   false, // Start as inactive
	}

	if err := registry.Register(currency); err != nil {
		log.Fatal(err)
	}

	// Initially not supported (inactive)
	fmt.Printf("Is TEST supported? %t\n", registry.IsSupported("TEST"))

	// Activate the currency
	if err := registry.Activate("TEST"); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("After activation, is TEST supported? %t\n", registry.IsSupported("TEST"))

	// Deactivate the currency
	if err := registry.Deactivate("TEST"); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("After deactivation, is TEST supported? %t\n", registry.IsSupported("TEST"))

	// Unregister the currency
	if err := registry.Unregister("TEST"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Currency unregistered")
}

// ExampleValidation demonstrates currency validation
func ExampleValidation() {
	// Valid currency
	validCurrency := CurrencyMeta{
		Code:     "USD",
		Name:     "US Dollar",
		Symbol:   "$",
		Decimals: 2,
	}

	if err := Register(validCurrency); err != nil {
		fmt.Printf("Valid currency registration failed: %v\n", err)
	} else {
		fmt.Println("Valid currency registered successfully")
	}

	// Invalid currency code
	invalidCode := CurrencyMeta{
		Code:     "usd", // lowercase is invalid
		Name:     "US Dollar",
		Symbol:   "$",
		Decimals: 2,
	}

	if err := Register(invalidCode); err != nil {
		fmt.Printf("Invalid currency code rejected: %v\n", err)
	}

	// Invalid decimals
	invalidDecimals := CurrencyMeta{
		Code:     "USD",
		Name:     "US Dollar",
		Symbol:   "$",
		Decimals: 9, // too high
	}

	if err := Register(invalidDecimals); err != nil {
		fmt.Printf("Invalid decimals rejected: %v\n", err)
	}

	// Invalid symbol
	invalidSymbol := CurrencyMeta{
		Code:     "USD",
		Name:     "US Dollar",
		Symbol:   "", // empty symbol
		Decimals: 2,
	}

	if err := Register(invalidSymbol); err != nil {
		fmt.Printf("Invalid symbol rejected: %v\n", err)
	}
}

// ExampleBackwardCompatibility demonstrates legacy API usage
func ExampleBackwardCompatibility() {
	// Legacy registration (still works)
	RegisterLegacy("LEGACY", CurrencyMeta{
		Symbol:   "L",
		Decimals: 2,
	})

	// Legacy get (returns default for non-existent)
	meta := GetLegacy("LEGACY")
	fmt.Printf("Legacy currency: %s (%s)\n", meta.Code, meta.Symbol)

	// Legacy check
	if IsSupportedLegacy("USD") {
		fmt.Println("USD is supported (legacy check)")
	}

	// Legacy list
	codes := ListSupportedLegacy()
	fmt.Printf("Legacy supported codes: %v\n", codes)

	// Legacy count
	count := CountLegacy()
	fmt.Printf("Legacy count: %d\n", count)
}

// ExampleCurrencyMetadata demonstrates working with currency metadata
func ExampleCurrencyMetadata() {
	ctx := context.Background()
	registry, err := NewCurrencyRegistry(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Register currency with rich metadata
	richCurrency := CurrencyMeta{
		Code:     "GOLD",
		Name:     "Gold Standard",
		Symbol:   "Au",
		Decimals: 4,
		Country:  "Global",
		Region:   "Precious Metals",
		Active:   true,
		Metadata: map[string]string{
			"type":          "commodity",
			"atomic_number": "79",
			"atomic_weight": "196.967",
			"melting_point": "1064.18°C",
			"density":       "19.32 g/cm³",
			"exchange_type": "spot",
		},
	}

	if err := registry.Register(richCurrency); err != nil {
		log.Fatal(err)
	}

	// Retrieve and display metadata
	retrieved, err := registry.Get("GOLD")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Gold currency metadata:\n")
	fmt.Printf("- Type: %s\n", retrieved.Metadata["type"])
	fmt.Printf("- Atomic Number: %s\n", retrieved.Metadata["atomic_number"])
	fmt.Printf("- Melting Point: %s\n", retrieved.Metadata["melting_point"])
	fmt.Printf("- Exchange Type: %s\n", retrieved.Metadata["exchange_type"])
}

// ExampleCurrencyStatistics demonstrates getting currency statistics
func ExampleCurrencyStatistics() {
	ctx := context.Background()
	registry, err := NewCurrencyRegistry(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Get counts
	total, err := registry.Count()
	if err != nil {
		log.Fatal(err)
	}

	active, err := registry.CountActive()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Currency Statistics:\n")
	fmt.Printf("- Total currencies: %d\n", total)
	fmt.Printf("- Active currencies: %d\n", active)
	fmt.Printf("- Inactive currencies: %d\n", total-active)

	// List all currencies with details
	all, err := registry.ListAll()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nAll currencies:\n")
	for _, currency := range all {
		status := "Active"
		if !currency.Active {
			status = "Inactive"
		}
		fmt.Printf("- %s (%s): %s - %d decimals\n",
			currency.Code, currency.Symbol, status, currency.Decimals)
	}
}

// ExampleCurrencyEvents demonstrates working with currency events
func ExampleCurrencyEvents() {
	ctx := context.Background()
	registry, err := NewCurrencyRegistry(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// The enhanced registry automatically publishes events for:
	// - Entity registration
	// - Entity updates
	// - Entity unregistration
	// - Entity activation/deactivation

	// Register a currency (triggers registration event)
	eventCurrency := CurrencyMeta{
		Code:     "EVENT",
		Name:     "Event Currency",
		Symbol:   "E",
		Decimals: 2,
	}

	if err := registry.Register(eventCurrency); err != nil {
		log.Fatal(err)
	}

	// Activate the currency (triggers activation event)
	if err := registry.Activate("EVENT"); err != nil {
		log.Fatal(err)
	}

	// Deactivate the currency (triggers deactivation event)
	if err := registry.Deactivate("EVENT"); err != nil {
		log.Fatal(err)
	}

	// Unregister the currency (triggers unregistration event)
	if err := registry.Unregister("EVENT"); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Currency events processed successfully")
}

// ExampleCurrencyCaching demonstrates the caching behavior
func ExampleCurrencyCaching() {
	ctx := context.Background()
	registry, err := NewCurrencyRegistry(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// First get (cache miss)
	start := time.Now()
	_, err = registry.Get("USD")
	if err != nil {
		log.Fatal(err)
	}
	firstGet := time.Since(start)

	// Second get (cache hit)
	start = time.Now()
	_, err = registry.Get("USD")
	if err != nil {
		log.Fatal(err)
	}
	secondGet := time.Since(start)

	fmt.Printf("Cache performance:\n")
	fmt.Printf("- First get (cache miss): %v\n", firstGet)
	fmt.Printf("- Second get (cache hit): %v\n", secondGet)
	fmt.Printf("- Cache improvement: %v\n", firstGet-secondGet)
}

// ExampleCurrencyHealth demonstrates health checking
func ExampleCurrencyHealth() {
	ctx := context.Background()
	_, err := NewCurrencyRegistry(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// The enhanced registry provides health monitoring
	// This includes:
	// - Registry availability
	// - Cache health
	// - Persistence health (if enabled)
	// - Error rates
	// - Performance metrics

	fmt.Println("Currency registry health monitoring enabled")
	fmt.Println("Health checks include:")
	fmt.Println("- Registry availability")
	fmt.Println("- Cache performance")
	fmt.Println("- Error rates")
	fmt.Println("- Response times")
}
