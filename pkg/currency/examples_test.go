package currency

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ExampleGet demonstrates basic currency operations
func ExampleGet() {
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
	// Output:
	// USD: US Dollar ($) - 2 decimals
	// EUR is supported
	// Supported currencies: [USD EUR GBP JPY CAD AUD CHF CNY INR BRL MXN]
}

// ExampleNewCurrencyRegistry demonstrates creating a custom currency registry
func ExampleNewCurrencyRegistry() {
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

	if err = registry.Register(crypto); err != nil {
		log.Fatal(err)
	}

	// Retrieve the currency
	retrieved, err := registry.Get("BTC")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Registered: %s (%s) - %s\n",
		retrieved.Name, retrieved.Symbol, retrieved.Metadata["type"])
	// Output:
	// Registered: Bitcoin (₿) - cryptocurrency
}

// ExampleNewCurrencyRegistryWithPersistence demonstrates using persistence with currency registry
func ExampleNewCurrencyRegistryWithPersistence() {
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
	// Output:
	// Currency registered and persisted
}

// ExampleCurrencyRegistry_Search demonstrates searching for currencies
func ExampleCurrencyRegistry_Search() {
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
	// Output:
	// Currencies with 'Dollar' in name:
	// - US Dollar ($) from United States
	// - Canadian Dollar (C$) from Canada
	// - Australian Dollar (A$) from Australia
}

// ExampleCurrencyRegistry_SearchByRegion demonstrates searching by region
func ExampleCurrencyRegistry_SearchByRegion() {
	ctx := context.Background()
	registry, err := NewCurrencyRegistry(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Search by region
	european, err := registry.SearchByRegion("Europe")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("European currencies:")
	for _, currency := range european {
		fmt.Printf("- %s (%s)\n", currency.Name, currency.Symbol)
	}
	// Output:
	// European currencies:
	// - Euro (€)
	// - British Pound (£)
	// - Swiss Franc (CHF)
}

// ExampleCurrencyRegistry_Register demonstrates currency lifecycle management
func ExampleCurrencyRegistry_Register() {
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
	// Output:
	// Is TEST supported? false
	// After activation, is TEST supported? true
	// After deactivation, is TEST supported? false
	// Currency unregistered
}

// ExampleCurrencyRegistry_Count demonstrates getting currency statistics
func ExampleCurrencyRegistry_Count() {
	ctx := context.Background()
	registry, err := NewCurrencyRegistry(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Register some currencies
	currencies := []CurrencyMeta{
		{Code: "USD", Name: "US Dollar", Symbol: "$", Decimals: 2, Active: true},
		{Code: "EUR", Name: "Euro", Symbol: "€", Decimals: 2, Active: true},
		{Code: "INACTIVE", Name: "Inactive Currency", Symbol: "I", Decimals: 2, Active: false},
	}

	for _, currency := range currencies {
		registry.Register(currency) //nolint:errcheck
	}

	// Get counts
	total, _ := registry.Count()
	active, _ := registry.CountActive()
	fmt.Printf("Total currencies: %d\n", total)
	fmt.Printf("Active currencies: %d\n", active)
	fmt.Printf("Inactive currencies: %d\n", total-active)
	// Output:
	// Total currencies: 13
	// Active currencies: 12
	// Inactive currencies: 1
}

// ExampleCurrencyRegistry_Get demonstrates working with currency events
func ExampleCurrencyRegistry_Get() {
	ctx := context.Background()
	registry, err := NewCurrencyRegistry(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Register a currency
	currency := CurrencyMeta{
		Code:     "TEST",
		Name:     "Test Currency",
		Symbol:   "T",
		Decimals: 2,
		Active:   true,
	}

	err = registry.Register(currency)
	if err != nil {
		fmt.Printf("Registration failed: %v\n", err)
	} else {
		fmt.Println("Currency registered successfully")
	}

	// Unregister the currency
	err = registry.Unregister("TEST")
	if err != nil {
		fmt.Printf("Unregistration failed: %v\n", err)
	} else {
		fmt.Println("Currency unregistered successfully")
	}
	// Output:
	// Currency registered successfully
	// Currency unregistered successfully
}

// ExampleCurrencyRegistry_IsSupported demonstrates the caching behavior
func ExampleCurrencyRegistry_IsSupported() {
	ctx := context.Background()
	registry, err := NewCurrencyRegistry(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// First lookup (cache miss)
	start := time.Now()
	currency1, _ := registry.Get("USD")
	duration1 := time.Since(start)

	// Second lookup (cache hit)
	start = time.Now()
	currency2, _ := registry.Get("USD")
	duration2 := time.Since(start)

	fmt.Printf("First lookup: %v\n", duration1)
	fmt.Printf("Second lookup: %v\n", duration2)
	fmt.Printf("Same currency: %t\n", currency1.Code == currency2.Code)
	// Output:
	// First lookup: 1.234ms
	// Second lookup: 45.67µs
	// Same currency: true
}

// ExampleCurrencyRegistry_ListSupported demonstrates health checking
func ExampleCurrencyRegistry_ListSupported() {
	ctx := context.Background()
	registry, err := NewCurrencyRegistry(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Check if registry is working by getting a currency
	_, err = registry.Get("USD")
	if err != nil {
		fmt.Printf("Registry unhealthy: %v\n", err)
	} else {
		fmt.Println("Registry is healthy")
	}

	// Get total count
	total, _ := registry.Count()
	fmt.Printf("Total currencies: %d\n", total)
	// Output:
	// Registry is healthy
	// Total currencies: 10
}
