package service

import (
	"context"
	"slices"
	"testing"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurrencyService(t *testing.T) {
	ctx := context.Background()

	t.Run("create currency service", func(t *testing.T) {
		registry, err := currency.NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		service := NewCurrencyService(registry)
		assert.NotNil(t, service)
		assert.Equal(t, registry, service.GetRegistry())
	})

	t.Run("get currency", func(t *testing.T) {
		registry, err := currency.NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		service := NewCurrencyService(registry)

		// Test getting existing currency
		usd, err := service.GetCurrency(ctx, "USD")
		require.NoError(t, err)
		assert.Equal(t, "USD", usd.Code)
		assert.Equal(t, "US Dollar", usd.Name)
		assert.Equal(t, "$", usd.Symbol)
		assert.Equal(t, 2, usd.Decimals)

		// Test getting non-existent currency
		_, err = service.GetCurrency(ctx, "INVALID")
		assert.Error(t, err)
	})

	t.Run("list supported currencies", func(t *testing.T) {
		registry, err := currency.NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		service := NewCurrencyService(registry)

		supported, err := service.ListSupportedCurrencies(ctx)
		require.NoError(t, err)
		assert.Greater(t, len(supported), 0)

		// Check that USD is in the list
		found := slices.Contains(supported, "USD")
		assert.True(t, found)
	})

	t.Run("list all currencies", func(t *testing.T) {
		registry, err := currency.NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		service := NewCurrencyService(registry)

		all, err := service.ListAllCurrencies(ctx)
		require.NoError(t, err)
		assert.Greater(t, len(all), 0)

		// Check that we have currency metadata
		found := false
		for _, currency := range all {
			if currency.Code == "USD" {
				found = true
				assert.Equal(t, "US Dollar", currency.Name)
				assert.Equal(t, "$", currency.Symbol)
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("register currency", func(t *testing.T) {
		registry, err := currency.NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		service := NewCurrencyService(registry)

		newCurrency := currency.CurrencyMeta{
			Code:     "TST",
			Name:     "Test Currency",
			Symbol:   "T",
			Decimals: 2,
			Country:  "Test Country",
			Region:   "Test Region",
			Active:   true,
		}

		err = service.RegisterCurrency(ctx, newCurrency)
		require.NoError(t, err)

		// Verify registration
		retrieved, err := service.GetCurrency(ctx, "TST")
		require.NoError(t, err)
		assert.Equal(t, "TST", retrieved.Code)
		assert.Equal(t, "Test Currency", retrieved.Name)
		assert.Equal(t, "T", retrieved.Symbol)
		assert.Equal(t, 2, retrieved.Decimals)
	})

	t.Run("unregister currency", func(t *testing.T) {
		registry, err := currency.NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		service := NewCurrencyService(registry)

		// Register a test currency first
		testCurrency := currency.CurrencyMeta{
			Code:     "TSU",
			Name:     "Test Currency 2",
			Symbol:   "T2",
			Decimals: 2,
		}

		err = service.RegisterCurrency(ctx, testCurrency)
		require.NoError(t, err)

		// Verify it exists
		_, err = service.GetCurrency(ctx, "TSU")
		require.NoError(t, err)

		// Unregister it
		err = service.UnregisterCurrency(ctx, "TSU")
		require.NoError(t, err)

		// Verify it's gone
		_, err = service.GetCurrency(ctx, "TSU")
		assert.Error(t, err)
	})

	t.Run("activate and deactivate currency", func(t *testing.T) {
		registry, err := currency.NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		service := NewCurrencyService(registry)

		// Register a currency as inactive
		currency := currency.CurrencyMeta{
			Code:     "TSV",
			Name:     "Test Currency 3",
			Symbol:   "T3",
			Decimals: 2,
			Active:   false,
		}

		err = service.RegisterCurrency(ctx, currency)
		require.NoError(t, err)

		// Initially should not be supported (inactive)
		assert.False(t, service.IsCurrencySupported(ctx, "TSV"))

		// Activate
		err = service.ActivateCurrency(ctx, "TSV")
		require.NoError(t, err)
		assert.True(t, service.IsCurrencySupported(ctx, "TSV"))

		// Deactivate
		err = service.DeactivateCurrency(ctx, "TSV")
		require.NoError(t, err)
		assert.False(t, service.IsCurrencySupported(ctx, "TSV"))
	})

	t.Run("search currencies", func(t *testing.T) {
		registry, err := currency.NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		service := NewCurrencyService(registry)

		// Search for "Dollar"
		results, err := service.SearchCurrencies(ctx, "Dollar")
		require.NoError(t, err)
		assert.Greater(t, len(results), 0)

		// Check that USD is in results
		found := false
		for _, currency := range results {
			if currency.Code == "USD" {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("search currencies by region", func(t *testing.T) {
		registry, err := currency.NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		service := NewCurrencyService(registry)

		// Search for North America
		results, err := service.SearchCurrenciesByRegion(ctx, "North America")
		require.NoError(t, err)
		assert.Greater(t, len(results), 0)

		// Check that USD is in results
		found := false
		for _, currency := range results {
			if currency.Code == "USD" {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("get currency statistics", func(t *testing.T) {
		registry, err := currency.NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		service := NewCurrencyService(registry)

		stats, err := service.GetCurrencyStatistics(ctx)
		require.NoError(t, err)
		assert.NotNil(t, stats)

		// Check that we have the expected fields
		assert.Contains(t, stats, "total_currencies")
		assert.Contains(t, stats, "active_currencies")
		assert.Contains(t, stats, "inactive_currencies")

		// Check that values are reasonable
		total := stats["total_currencies"].(int)
		active := stats["active_currencies"].(int)
		inactive := stats["inactive_currencies"].(int)

		assert.Greater(t, total, 0)
		assert.Greater(t, active, 0)
		assert.LessOrEqual(t, active, total)
		assert.Equal(t, total-active, inactive)
	})

	t.Run("validate currency code", func(t *testing.T) {
		registry, err := currency.NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		service := NewCurrencyService(registry)

		// Valid codes
		validCodes := []string{"USD", "EUR", "GBP", "JPY", "CAD"}
		for _, code := range validCodes {
			err := service.ValidateCurrencyCode(ctx, code)
			assert.NoError(t, err, "Code %s should be valid", code)
		}

		// Invalid codes
		invalidCodes := []string{"usd", "US", "USDD", "123", "USD1", ""}
		for _, code := range invalidCodes {
			err := service.ValidateCurrencyCode(ctx, code)
			assert.Error(t, err, "Code %s should be invalid", code)
		}
	})

	t.Run("get default currency", func(t *testing.T) {
		registry, err := currency.NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		service := NewCurrencyService(registry)

		defaultCurrency, err := service.GetDefaultCurrency(ctx)
		require.NoError(t, err)
		assert.Equal(t, "USD", defaultCurrency.Code)
		assert.Equal(t, "US Dollar", defaultCurrency.Name)
		assert.Equal(t, "$", defaultCurrency.Symbol)
		assert.Equal(t, 2, defaultCurrency.Decimals)
	})

	t.Run("is currency supported", func(t *testing.T) {
		registry, err := currency.NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		service := NewCurrencyService(registry)

		// Test supported currency
		assert.True(t, service.IsCurrencySupported(ctx, "USD"))

		// Test unsupported currency
		assert.False(t, service.IsCurrencySupported(ctx, "INVALID"))

		// Test invalid format
		assert.False(t, service.IsCurrencySupported(ctx, "usd"))
	})
}
