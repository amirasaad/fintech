package currency_test

import (
	"context"
	"slices"
	"testing"

	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"
	currencysvc "github.com/amirasaad/fintech/pkg/service/currency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurrencyService(t *testing.T) {
	ctx := context.Background()

	require := require.New(t)
	assert := assert.New(t)

	t.Run("create currency service", func(t *testing.T) {
		registry, err := currency.New(ctx)
		require.NoError(err)

		service := currencysvc.NewService(registry, slog.Default())
		assert.NotNil(service)
		assert.Equal(registry, service.GetRegistry())
	})

	t.Run("get currency", func(t *testing.T) {
		registry, err := currency.New(ctx)
		require.NoError(err)

		service := currencysvc.NewService(registry, slog.Default())

		// Test getting existing currency
		usd, err := service.GetCurrency(ctx, "USD")
		require.NoError(err)
		assert.Equal("USD", usd.Code)
		assert.Equal("US Dollar", usd.Name)
		assert.Equal("$", usd.Symbol)
		assert.Equal(2, usd.Decimals)

		// Test getting non-existent currency
		_, err = service.GetCurrency(ctx, "INVALID")
		require.Error(err)
	})

	t.Run("list supported currencies", func(t *testing.T) {
		registry, err := currency.New(ctx)
		require.NoError(err)

		service := currencysvc.NewService(registry, slog.Default())

		supported, err := service.ListSupportedCurrencies(ctx)
		require.NoError(err)
		assert.NotEmpty(supported)

		// Check that USD is in the list
		found := slices.Contains(supported, "USD")
		assert.True(found)
	})

	t.Run("list all currencies", func(t *testing.T) {
		registry, err := currency.New(ctx)
		require.NoError(err)

		service := currencysvc.NewService(registry, slog.Default())

		all, err := service.ListAllCurrencies(ctx)
		require.NoError(err)
		assert.NotEmpty(all)

		// Check that we have currency metadata
		found := false
		for _, currency := range all {
			if currency.Code == "USD" {
				found = true
				assert.Equal("US Dollar", currency.Name)
				assert.Equal("$", currency.Symbol)
				break
			}
		}
		assert.True(found)
	})

	t.Run("register currency", func(t *testing.T) {
		registry, err := currency.New(ctx)
		require.NoError(err)

		service := currencysvc.NewService(registry, slog.Default())

		newCurrency := currency.Meta{
			Code:     "TST",
			Name:     "Test Currency",
			Symbol:   "T",
			Decimals: 2,
			Country:  "Test Country",
			Region:   "Test Region",
			Active:   true,
		}

		err = service.RegisterCurrency(ctx, newCurrency)
		require.NoError(err)

		// Verify registration
		retrieved, err := service.GetCurrency(ctx, "TST")
		require.NoError(err)
		assert.Equal("TST", retrieved.Code)
		assert.Equal("Test Currency", retrieved.Name)
		assert.Equal("T", retrieved.Symbol)
		assert.Equal(2, retrieved.Decimals)
	})

	t.Run("unregister currency", func(t *testing.T) {
		registry, err := currency.New(ctx)
		require.NoError(err)

		service := currencysvc.NewService(registry, slog.Default())

		// Register a test currency first
		testCurrency := currency.Meta{
			Code:     "TSU",
			Name:     "Test Currency 2",
			Symbol:   "T2",
			Decimals: 2,
		}

		err = service.RegisterCurrency(ctx, testCurrency)
		require.NoError(err)

		// Verify it exists
		_, err = service.GetCurrency(ctx, "TSU")
		require.NoError(err)

		// Unregister it
		err = service.UnregisterCurrency(ctx, "TSU")
		require.NoError(err)

		// Verify it's gone
		_, err = service.GetCurrency(ctx, "TSU")
		require.Error(err)
	})

	t.Run("activate and deactivate currency", func(t *testing.T) {
		registry, err := currency.New(ctx)
		require.NoError(err)

		service := currencysvc.NewService(registry, slog.Default())

		// Register a currency as inactive
		currency := currency.Meta{
			Code:     "TSV",
			Name:     "Test Currency 3",
			Symbol:   "T3",
			Decimals: 2,
			Active:   false,
		}

		err = service.RegisterCurrency(ctx, currency)
		require.NoError(err)

		// Initially should not be supported (inactive)
		assert.False(service.IsCurrencySupported(ctx, "TSV"))

		// Activate
		err = service.ActivateCurrency(ctx, "TSV")
		require.NoError(err)
		assert.True(service.IsCurrencySupported(ctx, "TSV"))

		// Deactivate
		err = service.DeactivateCurrency(ctx, "TSV")
		require.NoError(err)
		assert.False(service.IsCurrencySupported(ctx, "TSV"))
	})

	t.Run("search currencies", func(t *testing.T) {
		registry, err := currency.New(ctx)
		require.NoError(err)

		service := currencysvc.NewService(registry, slog.Default())

		// Search for "Dollar"
		results, err := service.SearchCurrencies(ctx, "Dollar")
		require.NoError(err)
		assert.NotEmpty(results)

		// Check that USD is in results
		found := false
		for _, currency := range results {
			if currency.Code == "USD" {
				found = true
				break
			}
		}
		assert.True(found)
	})

	t.Run("search currencies by region", func(t *testing.T) {
		registry, err := currency.New(ctx)
		require.NoError(err)

		service := currencysvc.NewService(registry, slog.Default())

		// Search for North America
		results, err := service.SearchCurrenciesByRegion(ctx, "North America")
		require.NoError(err)
		assert.NotEmpty(results)

		// Check that USD is in results
		found := false
		for _, currency := range results {
			if currency.Code == "USD" {
				found = true
				break
			}
		}
		assert.True(found)
	})

	t.Run("get currency statistics", func(t *testing.T) {
		registry, err := currency.New(ctx)
		require.NoError(err)

		service := currencysvc.NewService(registry, slog.Default())

		stats, err := service.GetCurrencyStatistics(ctx)
		require.NoError(err)
		assert.NotNil(stats)

		// Check that we have the expected fields
		assert.Contains(stats, "total_currencies")
		assert.Contains(stats, "active_currencies")
		assert.Contains(stats, "inactive_currencies")

		// Check that values are reasonable
		total := stats["total_currencies"].(int)
		active := stats["active_currencies"].(int)
		inactive := stats["inactive_currencies"].(int)

		assert.Positive(total)
		assert.Positive(active)
		assert.LessOrEqual(active, total)
		assert.Equal(total-active, inactive)
	})

	t.Run("validate currency code", func(t *testing.T) {
		registry, err := currency.New(ctx)
		require.NoError(err)

		service := currencysvc.NewService(registry, slog.Default())

		// Valid codes
		validCodes := []string{"USD", "EUR", "GBP", "JPY", "CAD"}
		for _, code := range validCodes {
			err := service.ValidateCurrencyCode(ctx, code)
			require.NoError(err, "Code %s should be valid", code)
		}

		// Invalid codes
		invalidCodes := []string{"usd", "US", "USDD", "123", "USD1", ""}
		for _, code := range invalidCodes {
			err := service.ValidateCurrencyCode(ctx, code)
			require.Error(err, "Code %s should be invalid", code)
		}
	})

	t.Run("get default currency", func(t *testing.T) {
		registry, err := currency.New(ctx)
		require.NoError(err)

		service := currencysvc.NewService(registry, slog.Default())

		defaultCurrency, err := service.GetDefaultCurrency(ctx)
		require.NoError(err)
		assert.Equal("USD", defaultCurrency.Code)
		assert.Equal("US Dollar", defaultCurrency.Name)
		assert.Equal("$", defaultCurrency.Symbol)
		assert.Equal(2, defaultCurrency.Decimals)
	})

	t.Run("is currency supported", func(t *testing.T) {
		registry, err := currency.New(ctx)
		require.NoError(err)

		service := currencysvc.NewService(registry, slog.Default())

		// Test supported currency
		assert.True(service.IsCurrencySupported(ctx, "USD"))

		// Test unsupported currency
		assert.False(service.IsCurrencySupported(ctx, "INVALID"))

		// Test invalid format
		assert.False(service.IsCurrencySupported(ctx, "usd"))
	})
}
