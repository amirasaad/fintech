package currency_test

import (
	"context"
	"slices"
	"testing"

	"log/slog"

	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/registry"
	"github.com/amirasaad/fintech/pkg/service/currency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testCurrency = currency.Entity{
	Entity:   registry.NewBaseEntity("TST", "Test Currency"),
	Code:     "TST",
	Name:     "Test Currency",
	Symbol:   "T",
	Decimals: 2,
	Country:  "Test Country",
	Region:   "Test Region",
	Active:   true,
}

func TestCurrencyService(t *testing.T) {
	ctx := context.Background()

	require := require.New(t)
	assert := assert.New(t)

	setupTestService := func() *currency.Service {
		// Create a new registry
		reg := registry.NewEnhanced(registry.Config{
			Name: "test-registry",
		})

		// Create the service with the registry
		service := currency.New(reg, slog.Default())

		// Register default currencies needed for tests
		defaultCurrencies := []currency.Entity{
			{
				Code:     "USD",
				Name:     "US Dollar",
				Symbol:   "$",
				Decimals: 2,
				Country:  "United States",
				Region:   "North America",
				Active:   true,
			},
			{
				Code:     "EUR",
				Name:     "Euro",
				Symbol:   "€",
				Decimals: 2,
				Country:  "European Union",
				Region:   "Europe",
				Active:   true,
			},
		}

		// Register default currencies
		for _, curr := range defaultCurrencies {
			// Register the currency using the service's Register method
			err := service.Register(ctx, curr)
			if err != nil {
				t.Fatalf("failed to register test currency %s: %v", curr.Code, err)
			}
		}

		// Register test currency
		err := service.Register(ctx, testCurrency)
		if err != nil {
			t.Fatalf("failed to register test currency: %v", err)
		}

		return service
	}

	t.Run("get currency", func(t *testing.T) {
		service := setupTestService()

		// Test getting existing currency (USD is one of the default currencies)
		usd, err := service.Get(ctx, "USD")
		require.NoError(err)
		assert.Equal(money.Code("USD"), usd.Code)
		assert.Equal(2, usd.Decimals)

		// Test getting non-existent currency
		_, err = service.Get(ctx, "INVALID")
		require.Error(err)
	})

	t.Run("list supported currencies", func(t *testing.T) {
		service := setupTestService()

		supported, err := service.ListSupported(ctx)
		require.NoError(err)
		assert.NotEmpty(supported)

		// Check that USD is in the list
		found := slices.Contains(supported, "USD")
		assert.True(found)
	})

	t.Run("list all currencies", func(t *testing.T) {
		service := setupTestService()

		// Get all currencies
		currencies, err := service.ListAll(ctx)
		require.NoError(err)

		// Should have at least the default currencies
		require.GreaterOrEqual(len(currencies), 2)

		// Verify we can find USD and EUR
		hasUSD := false
		hasEUR := false
		for _, c := range currencies {
			if c.Code == "USD" {
				hasUSD = true
			}
			if c.Code == "EUR" {
				hasEUR = true
			}
		}

		assert.True(hasUSD, "Should have USD")
		assert.True(hasEUR, "Should have EUR")
	})

	t.Run("register currency", func(t *testing.T) {
		service := setupTestService()

		err := service.Register(ctx, testCurrency)
		require.NoError(err)

		// Verify registration
		retrieved, err := service.Get(ctx, "TST")
		require.NoError(err)
		assert.Equal(money.Code("TST"), retrieved.Code)
		assert.Equal(2, retrieved.Decimals)
	})

	t.Run("unregister currency", func(t *testing.T) {
		service := setupTestService()

		// Test registering a new currency
		err := service.Register(ctx, currency.Entity{
			Code:     "BTC",
			Name:     "Bitcoin",
			Symbol:   "₿",
			Decimals: 8,
			Country:  "Global",
			Region:   "Global",
			Active:   true,
		})
		require.NoError(err)

		// Verify it's registered
		_, err = service.Get(ctx, "BTC")
		require.NoError(err)

		// Now unregister it
		err = service.Unregister(ctx, "BTC")
		require.NoError(err)

		// Verify it's gone
		_, err = service.Get(ctx, "BTC")
		require.Error(err)
		assert.Contains(err.Error(), "not found")
	})

	t.Run("activate and deactivate currency", func(t *testing.T) {
		service := setupTestService()

		// First, ensure the test currency doesn't exist
		_, err := service.Get(ctx, "TSV")
		if err == nil {
			// If it exists, unregister it first
			err = service.Unregister(ctx, "TSV")
			require.NoError(err)
		}

		// Register a new currency (should be active by default)
		err = service.Register(ctx, currency.Entity{
			Code:     "TSV",
			Name:     "Test Currency 3",
			Symbol:   "T3",
			Decimals: 2,
			Country:  "Test Country",
			Region:   "Test Region",
			Active:   false, // Explicitly set as inactive
		})
		require.NoError(err, "Failed to register test currency")

		// Check if the currency is registered but inactive
		_, err = service.Get(ctx, "TSV")
		require.NoError(err, "Currency should be registered")

		// Initially should not be supported (inactive)
		isSupported := service.IsSupported(ctx, "TSV")
		assert.False(isSupported, "Inactive currency should not be supported")

		// Activate
		err = service.Activate(ctx, "TSV")
		require.NoError(err, "Failed to activate currency")

		// Now should be supported
		isSupported = service.IsSupported(ctx, "TSV")
		assert.True(isSupported, "Active currency should be supported")

		// Deactivate
		err = service.Deactivate(ctx, "TSV")
		require.NoError(err, "Failed to deactivate currency")

		// Should no longer be supported
		isSupported = service.IsSupported(ctx, "TSV")
		assert.False(isSupported, "Deactivated currency should not be supported")

		// Clean up
		err = service.Unregister(ctx, "TSV")
		require.NoError(err, "Failed to clean up test currency")
	})

	t.Run("search currencies", func(t *testing.T) {
		service := setupTestService()

		// Search for "dollar" should find USD
		results, err := service.Search(ctx, "dollar")
		require.NoError(err)

		foundUSD := false
		for _, c := range results {
			if c.Code == "USD" {
				foundUSD = true
				break
			}
		}
		assert.True(foundUSD, "Should find USD when searching for 'dollar'")
	})

	t.Run("search currencies by region", func(t *testing.T) {
		service := setupTestService()

		// Search for North America
		results, err := service.SearchByRegion(ctx, "North America")
		require.NoError(err)

		foundUSD := false
		for _, c := range results {
			if c.Code == "USD" {
				foundUSD = true
				break
			}
		}
		assert.True(foundUSD, "Should find USD when searching for 'North America'")
	})

	t.Run("get currency statistics", func(t *testing.T) {
		service := setupTestService()

		stats, err := service.GetStatistics(ctx)
		require.NoError(err)
		assert.NotNil(stats)

		// Check that we have the expected fields
		assert.Contains(stats, "total_currencies")
		assert.Contains(stats, "active_currencies")

		total := int(stats["total_currencies"].(int))
		active := int(stats["active_currencies"].(int))

		assert.Positive(total)
		assert.Positive(active)
		assert.LessOrEqual(active, total)
	})

	t.Run("validate currency code", func(t *testing.T) {
		service := setupTestService()

		// Valid codes
		validCodes := []string{"USD", "EUR", "GBP", "JPY", "CAD"}
		for _, code := range validCodes {
			err := service.ValidateCode(ctx, code)
			require.NoError(err, "Code %s should be valid", code)
		}

		// Invalid codes
		invalidCodes := []string{"usd", "US", "USDD", "123", "USD1", ""}
		for _, code := range invalidCodes {
			err := service.ValidateCode(ctx, code)
			require.Error(err, "Code %s should be invalid", code)
		}
	})

	t.Run("get default currency", func(t *testing.T) {
		service := setupTestService()

		// Get the default currency (USD)
		defaultCurrency, err := service.GetDefault(ctx)
		require.NoError(err)

		// Should be USD
		assert.Equal(money.Code("USD"), defaultCurrency.Code)
		assert.Equal(2, defaultCurrency.Decimals)
	})

	t.Run("is currency supported", func(t *testing.T) {
		service := setupTestService()

		// Test with default currencies
		assert.True(service.IsSupported(ctx, "USD"))
		assert.True(service.IsSupported(ctx, "EUR"))

		// Test with non-existent currency
		assert.False(service.IsSupported(ctx, "NONEXISTENT"))

		// Test invalid format
		assert.False(service.IsSupported(ctx, "usd"))
	})
}
