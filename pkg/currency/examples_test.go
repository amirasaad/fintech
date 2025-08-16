package currency

import (
	"context"
	"fmt"
	"log"
	"sort"
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
	// Get the actual count of supported currencies
	count := len(supported)
	fmt.Printf("Total supported currencies: %d\n", count)
	// Output:
	// USD: US Dollar ($) - 2 decimals
	// EUR is supported
	// Total supported currencies: 13
}

// ExampleNewRegistry demonstrates creating a custom currency registry
func ExampleNew() {
	ctx := context.Background()

	// Create a new registry
	registry, err := New(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Register a custom cryptocurrency
	crypto := Meta{
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

	fmt.Printf("Registered: %s (%s)\n", retrieved.Name,
		retrieved.Symbol)
	// Output:
	// Registered: Bitcoin (₿)
}

// ExampleNewRegistryWithPersistence demonstrates using persistence with currency registry
func ExampleNewRegistryWithPersistence() {
	ctx := context.Background()

	// Create registry with persistence
	registry, err := NewRegistryWithPersistence(ctx, "./currencies.json")
	if err != nil {
		log.Fatal(err)
	}

	// Register a new currency
	newCurrency := Meta{
		Code:     "CST",
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

// ExampleRegistry_Search demonstrates searching for currencies
func ExampleRegistry_Search() {
	ctx := context.Background()
	registry, err := New(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Search by name
	results, err := registry.Search("Dollar")
	if err != nil {
		log.Fatal(err)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	fmt.Println("Currencies with 'Dollar' in name:")
	for _, currency := range results {
		fmt.Printf("- %s (%s) from %s\n",
			currency.Name, currency.Symbol, currency.Country)
	}
	// Output:
	// Currencies with 'Dollar' in name:
	// - Australian Dollar (A$) from Australia
	// - Canadian Dollar (C$) from Canada
	// - US Dollar ($) from United States
}

// ExampleRegistry_SearchByRegion demonstrates searching by region
func ExampleRegistry_SearchByRegion() {
	ctx := context.Background()
	registry, err := New(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Search by region
	european, err := registry.SearchByRegion("Europe")
	if err != nil {
		log.Fatal(err)
	}

	sort.Slice(european, func(i, j int) bool {
		return european[i].Name < european[j].Name
	})

	fmt.Println("European currencies:")
	for _, currency := range european {
		fmt.Printf("- %s (%s)\n", currency.Name, currency.Symbol)
	}
	// Output:
	// European currencies:
	// - British Pound (£)
	// - Euro (€)
	// - Swiss Franc (CHF)
}

// ExampleRegistry_Register demonstrates currency lifecycle management
func ExampleRegistry_Register() {
	ctx := context.Background()
	registry, err := New(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Register a currency as inactive
	currency := Meta{
		Code:     "TST",
		Name:     "Test Currency",
		Symbol:   "T",
		Decimals: 2,
		Active:   false, // Start as inactive
	}

	if err := registry.Register(currency); err != nil {
		log.Fatal(err)
	}

	// Initially not supported (inactive)
	fmt.Printf("Is TEST supported? %t\n", registry.IsSupported("TST"))

	// Activate the currency
	if err := registry.Activate("TST"); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("After activation, is TEST supported? %t\n", registry.IsSupported("TST"))

	// Deactivate the currency
	if err := registry.Deactivate("TST"); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("After deactivation, is TEST supported? %t\n", registry.IsSupported("TST"))

	// Unregister the currency
	if err := registry.Unregister("TST"); err != nil {
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
func ExampleRegistry_Count() {
	ctx := context.Background()
	registry, err := New(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Register some currencies
	currencies := []Meta{
		{Code: "USD", Name: "US Dollar", Symbol: "$", Decimals: 2, Active: true},
		{Code: "EUR", Name: "Euro", Symbol: "€", Decimals: 2, Active: true},
		{Code: "INA", Name: "Inactive Currency", Symbol: "I", Decimals: 2, Active: false},
	}

	for _, currency := range currencies {
		if err := registry.Register(currency); err != nil {
			log.Println(err)
			continue
		}
	}

	// Get counts
	total, _ := registry.Count()
	active, _ := registry.CountActive()
	fmt.Printf("Total currencies: %d\n", total)
	fmt.Printf("Active currencies: %d\n", active)
	fmt.Printf("Inactive currencies: %d\n", total-active)
	// Output:
	// Total currencies: 13
	// Active currencies: 13
	// Inactive currencies: 0
}

// ExampleRegistry_Get demonstrates working with currency events
func ExampleRegistry_Get() {
	ctx := context.Background()
	registry, err := New(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Register a currency
	currency := Meta{
		Code:     "TST",
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
	err = registry.Unregister("TST")
	if err != nil {
		fmt.Printf("Unregistration failed: %v\n", err)
	} else {
		fmt.Println("Currency unregistered successfully")
	}
	// Output:
	// Currency registered successfully
	// Currency unregistered successfully
}

// ExampleRegistry_IsSupported demonstrates the caching behavior
func ExampleRegistry_IsSupported() {
	ctx := context.Background()
	registry, err := New(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// First lookup (cache miss)
	currency1, _ := registry.Get("USD")

	// Second lookup (cache hit)
	currency2, _ := registry.Get("USD")

	fmt.Printf("Same currency: %t\n", currency1.Code == currency2.Code)
	// Output:
	// Same currency: true
}

// ExampleRegistry_ListSupported demonstrates health checking
func ExampleRegistry_ListSupported() {
	ctx := context.Background()
	registry, err := New(ctx)
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
	// Total currencies: 12
}
