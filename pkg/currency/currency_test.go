package currency

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurrencyEntity(t *testing.T) {
	tests := []struct {
		name     string
		meta     CurrencyMeta
		expected CurrencyMeta
	}{
		{
			name: "valid currency entity",
			meta: CurrencyMeta{
				Code:     "USD",
				Name:     "US Dollar",
				Symbol:   "$",
				Decimals: 2,
				Country:  "United States",
				Region:   "North America",
				Active:   true,
			},
			expected: CurrencyMeta{
				Code:     "USD",
				Name:     "US Dollar",
				Symbol:   "$",
				Decimals: 2,
				Country:  "United States",
				Region:   "North America",
				Active:   true,
			},
		},
		{
			name: "currency with metadata",
			meta: CurrencyMeta{
				Code:     "EUR",
				Name:     "Euro",
				Symbol:   "€",
				Decimals: 2,
				Country:  "European Union",
				Region:   "Europe",
				Active:   true,
				Metadata: map[string]string{
					"iso_code": "EUR",
					"type":     "fiat",
				},
			},
			expected: CurrencyMeta{
				Code:     "EUR",
				Name:     "Euro",
				Symbol:   "€",
				Decimals: 2,
				Country:  "European Union",
				Region:   "Europe",
				Active:   true,
				Metadata: map[string]string{
					"iso_code": "EUR",
					"type":     "fiat",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity := NewCurrencyEntity(tt.meta)

			assert.Equal(t, tt.expected.Code, entity.ID())
			assert.Equal(t, tt.expected.Name, entity.Name())
			assert.Equal(t, tt.expected.Active, entity.Active())
			assert.True(t, entity.CreatedAt().After(time.Time{}))
			assert.True(t, entity.UpdatedAt().After(time.Time{}))

			// Check metadata
			metadata := entity.Metadata()
			assert.Equal(t, tt.expected.Code, metadata["code"])
			assert.Equal(t, tt.expected.Symbol, metadata["symbol"])
			assert.Equal(t, "2", metadata["decimals"])
			assert.Equal(t, tt.expected.Country, metadata["country"])
			assert.Equal(t, tt.expected.Region, metadata["region"])
			assert.Equal(t, "true", metadata["active"])

			// Check custom metadata
			for k, v := range tt.expected.Metadata {
				assert.Equal(t, v, metadata[k])
			}

			// Check returned meta
			returnedMeta := entity.Meta()
			assert.Equal(t, tt.expected.Code, returnedMeta.Code)
			assert.Equal(t, tt.expected.Name, returnedMeta.Name)
			assert.Equal(t, tt.expected.Symbol, returnedMeta.Symbol)
			assert.Equal(t, tt.expected.Decimals, returnedMeta.Decimals)
			assert.Equal(t, tt.expected.Country, returnedMeta.Country)
			assert.Equal(t, tt.expected.Region, returnedMeta.Region)
			assert.Equal(t, tt.expected.Active, returnedMeta.Active)
		})
	}
}

func TestCurrencyValidator(t *testing.T) {
	validator := NewCurrencyValidator()
	ctx := context.Background()

	tests := []struct {
		name        string
		meta        CurrencyMeta
		expectError bool
		errorType   error
	}{
		{
			name: "valid currency",
			meta: CurrencyMeta{
				Code:     "USD",
				Name:     "US Dollar",
				Symbol:   "$",
				Decimals: 2,
			},
			expectError: false,
		},
		{
			name: "invalid currency code - lowercase",
			meta: CurrencyMeta{
				Code:     "usd",
				Name:     "US Dollar",
				Symbol:   "$",
				Decimals: 2,
			},
			expectError: true,
			errorType:   ErrInvalidCurrencyCode,
		},
		{
			name: "invalid currency code - too short",
			meta: CurrencyMeta{
				Code:     "US",
				Name:     "US Dollar",
				Symbol:   "$",
				Decimals: 2,
			},
			expectError: true,
			errorType:   ErrInvalidCurrencyCode,
		},
		{
			name: "invalid currency code - too long",
			meta: CurrencyMeta{
				Code:     "USDD",
				Name:     "US Dollar",
				Symbol:   "$",
				Decimals: 2,
			},
			expectError: true,
			errorType:   ErrInvalidCurrencyCode,
		},
		{
			name: "invalid decimals - negative",
			meta: CurrencyMeta{
				Code:     "USD",
				Name:     "US Dollar",
				Symbol:   "$",
				Decimals: -1,
			},
			expectError: true,
			errorType:   ErrInvalidDecimals,
		},
		{
			name: "invalid decimals - too high",
			meta: CurrencyMeta{
				Code:     "USD",
				Name:     "US Dollar",
				Symbol:   "$",
				Decimals: 9,
			},
			expectError: true,
			errorType:   ErrInvalidDecimals,
		},
		{
			name: "invalid symbol - empty",
			meta: CurrencyMeta{
				Code:     "USD",
				Name:     "US Dollar",
				Symbol:   "",
				Decimals: 2,
			},
			expectError: true,
			errorType:   ErrInvalidSymbol,
		},
		{
			name: "invalid symbol - too long",
			meta: CurrencyMeta{
				Code:     "USD",
				Name:     "US Dollar",
				Symbol:   "This symbol is way too long for a currency",
				Decimals: 2,
			},
			expectError: true,
			errorType:   ErrInvalidSymbol,
		},
		{
			name: "empty name",
			meta: CurrencyMeta{
				Code:     "USD",
				Name:     "",
				Symbol:   "$",
				Decimals: 2,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity := NewCurrencyEntity(tt.meta)
			err := validator.Validate(ctx, entity)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCurrencyValidatorMetadata(t *testing.T) {
	validator := NewCurrencyValidator()
	ctx := context.Background()

	tests := []struct {
		name        string
		metadata    map[string]string
		expectError bool
	}{
		{
			name: "valid metadata",
			metadata: map[string]string{
				"code":     "USD",
				"symbol":   "$",
				"decimals": "2",
			},
			expectError: false,
		},
		{
			name: "missing required field",
			metadata: map[string]string{
				"code":   "USD",
				"symbol": "$",
			},
			expectError: true,
		},
		{
			name: "invalid currency code in metadata",
			metadata: map[string]string{
				"code":     "usd",
				"symbol":   "$",
				"decimals": "2",
			},
			expectError: true,
		},
		{
			name: "invalid decimals in metadata",
			metadata: map[string]string{
				"code":     "USD",
				"symbol":   "$",
				"decimals": "invalid",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateMetadata(ctx, tt.metadata)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCurrencyRegistry(t *testing.T) {
	ctx := context.Background()

	t.Run("create registry", func(t *testing.T) {
		registry, err := NewCurrencyRegistry(ctx)
		require.NoError(t, err)
		assert.NotNil(t, registry)

		// Check that default currencies are registered
		count, err := registry.Count()
		require.NoError(t, err)
		assert.Greater(t, count, 0)

		// Check specific currencies
		usd, err := registry.Get("USD")
		require.NoError(t, err)
		assert.Equal(t, "USD", usd.Code)
		assert.Equal(t, "US Dollar", usd.Name)
		assert.Equal(t, "$", usd.Symbol)
		assert.Equal(t, 2, usd.Decimals)
		assert.True(t, usd.Active)
	})

	t.Run("register new currency", func(t *testing.T) {
		registry, err := NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		newCurrency := CurrencyMeta{
			Code:     "BTC",
			Name:     "Bitcoin",
			Symbol:   "₿",
			Decimals: 8,
			Country:  "Global",
			Region:   "Digital",
			Active:   true,
		}

		err = registry.Register(newCurrency)
		require.NoError(t, err)

		// Verify registration
		retrieved, err := registry.Get("BTC")
		require.NoError(t, err)
		assert.Equal(t, "BTC", retrieved.Code)
		assert.Equal(t, "Bitcoin", retrieved.Name)
		assert.Equal(t, "₿", retrieved.Symbol)
		assert.Equal(t, 8, retrieved.Decimals)
	})

	t.Run("register invalid currency", func(t *testing.T) {
		registry, err := NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		invalidCurrency := CurrencyMeta{
			Code:     "invalid",
			Name:     "Invalid Currency",
			Symbol:   "INV",
			Decimals: 2,
		}

		err = registry.Register(invalidCurrency)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCurrencyCode)
	})

	t.Run("get non-existent currency", func(t *testing.T) {
		registry, err := NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		_, err = registry.Get("NONEXISTENT")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "currency not found")
	})

	t.Run("list supported currencies", func(t *testing.T) {
		registry, err := NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		supported, err := registry.ListSupported()
		require.NoError(t, err)
		assert.Greater(t, len(supported), 0)

		// Check that USD is in the list
		found := false
		for _, code := range supported {
			if code == "USD" {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("list all currencies", func(t *testing.T) {
		registry, err := NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		all, err := registry.ListAll()
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

	t.Run("activate and deactivate currency", func(t *testing.T) {
		registry, err := NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		// Register a new currency
		newCurrency := CurrencyMeta{
			Code:     "TST",
			Name:     "Test Currency",
			Symbol:   "T",
			Decimals: 2,
			Active:   false,
		}

		err = registry.Register(newCurrency)
		require.NoError(t, err)

		// Initially should not be supported (inactive)
		assert.False(t, registry.IsSupported("TST"))

		// Activate
		err = registry.Activate("TST")
		require.NoError(t, err)
		assert.True(t, registry.IsSupported("TST"))

		// Deactivate
		err = registry.Deactivate("TST")
		require.NoError(t, err)
		assert.False(t, registry.IsSupported("TST"))
	})

	t.Run("unregister currency", func(t *testing.T) {
		registry, err := NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		// Register a test currency
		testCurrency := CurrencyMeta{
			Code:     "TSU",
			Name:     "Test Currency 2",
			Symbol:   "T2",
			Decimals: 2,
		}

		err = registry.Register(testCurrency)
		require.NoError(t, err)

		// Verify it exists
		_, err = registry.Get("TSU")
		require.NoError(t, err)

		// Unregister
		err = registry.Unregister("TSU")
		require.NoError(t, err)

		// Verify it's gone
		_, err = registry.Get("TSU")
		assert.Error(t, err)
	})

	t.Run("search currencies", func(t *testing.T) {
		registry, err := NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		// Search for "Dollar"
		results, err := registry.Search("Dollar")
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

	t.Run("search by region", func(t *testing.T) {
		registry, err := NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		// Search for North America
		results, err := registry.SearchByRegion("North America")
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

	t.Run("count currencies", func(t *testing.T) {
		registry, err := NewCurrencyRegistry(ctx)
		require.NoError(t, err)

		total, err := registry.Count()
		require.NoError(t, err)
		assert.Greater(t, total, 0)

		active, err := registry.CountActive()
		require.NoError(t, err)
		assert.Greater(t, active, 0)
		assert.LessOrEqual(t, active, total)
	})
}

func TestGlobalFunctions(t *testing.T) {
	t.Run("global get", func(t *testing.T) {
		usd, err := Get("USD")
		require.NoError(t, err)
		assert.Equal(t, "USD", usd.Code)
		assert.Equal(t, "US Dollar", usd.Name)
	})

	t.Run("global is supported", func(t *testing.T) {
		assert.True(t, IsSupported("USD"))
		assert.False(t, IsSupported("INVALID"))
	})

	t.Run("global list supported", func(t *testing.T) {
		supported, err := ListSupported()
		require.NoError(t, err)
		assert.Greater(t, len(supported), 0)

		// Check that USD is supported
		found := false
		for _, code := range supported {
			if code == "USD" {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("global count", func(t *testing.T) {
		count, err := Count()
		require.NoError(t, err)
		assert.Greater(t, count, 0)
	})

	t.Run("global search", func(t *testing.T) {
		results, err := Search("Dollar")
		require.NoError(t, err)
		assert.Greater(t, len(results), 0)
	})
}

func TestBackwardCompatibility(t *testing.T) {
	t.Run("legacy register", func(t *testing.T) {
		// This should not panic
		RegisterLegacy("LGY", CurrencyMeta{
			Symbol:   "L",
			Decimals: 2,
		})

		// Check that it was registered
		meta, err := Get("LGY")
		require.NoError(t, err)
		assert.Equal(t, "LGY", meta.Code)
		assert.Equal(t, "LGY", meta.Name)
		assert.Equal(t, "L", meta.Symbol)
		assert.Equal(t, 2, meta.Decimals)
		assert.True(t, meta.Active)
	})

	t.Run("legacy get", func(t *testing.T) {
		// Test with existing currency
		meta := GetLegacy("USD")
		assert.Equal(t, "USD", meta.Code)
		assert.Equal(t, "US Dollar", meta.Name)
		assert.True(t, meta.Active)

		// Test with non-existent currency
		meta = GetLegacy("NONEXISTENT")
		assert.Equal(t, "NONEXISTENT", meta.Code)
		assert.Equal(t, "NONEXISTENT", meta.Name)
		assert.False(t, meta.Active)
	})

	t.Run("legacy is supported", func(t *testing.T) {
		assert.True(t, IsSupportedLegacy("USD"))
		assert.False(t, IsSupportedLegacy("INVALID"))
	})

	t.Run("legacy list supported", func(t *testing.T) {
		codes := ListSupportedLegacy()
		assert.Greater(t, len(codes), 0)

		// Check that USD is in the list
		found := false
		for _, code := range codes {
			if code == "USD" {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("legacy unregister", func(t *testing.T) {
		// Register a test currency
		RegisterLegacy("TSV", CurrencyMeta{
			Symbol:   "T3",
			Decimals: 2,
		})

		// Unregister it
		success := UnregisterLegacy("TSV")
		assert.True(t, success)

		// Verify it's gone
		assert.False(t, IsSupportedLegacy("TSV"))
	})

	t.Run("legacy count", func(t *testing.T) {
		count := CountLegacy()
		assert.Greater(t, count, 0)
	})
}

func TestValidationHelpers(t *testing.T) {
	t.Run("isValidCurrencyCode", func(t *testing.T) {
		validCodes := []string{"USD", "EUR", "GBP", "JPY", "CAD"}
		for _, code := range validCodes {
			assert.True(t, isValidCurrencyCode(code), "Code %s should be valid", code)
		}

		invalidCodes := []string{"usd", "US", "USDD", "123", "USD1", ""}
		for _, code := range invalidCodes {
			assert.False(t, isValidCurrencyCode(code), "Code %s should be invalid", code)
		}
	})

	t.Run("validateCurrencyMeta", func(t *testing.T) {
		validMeta := CurrencyMeta{
			Code:     "USD",
			Name:     "US Dollar",
			Symbol:   "$",
			Decimals: 2,
		}
		assert.NoError(t, validateCurrencyMeta(validMeta))

		invalidMeta := CurrencyMeta{
			Code:     "invalid",
			Name:     "Invalid",
			Symbol:   "$",
			Decimals: 2,
		}
		assert.Error(t, validateCurrencyMeta(invalidMeta))
		assert.ErrorIs(t, validateCurrencyMeta(invalidMeta), ErrInvalidCurrencyCode)
	})
}
