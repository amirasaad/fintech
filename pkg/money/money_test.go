package money_test

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/money"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a new Money instance for testing
func mustNew(t *testing.T, amount float64, currency money.Code) *money.Money {
	t.Helper()
	m, err := money.New(amount, currency)
	require.NoError(t, err, "failed to create money for test")
	return m
}

// Helper function to create a new Money instance that might fail
func newMoney(t *testing.T, amount float64, currency money.Code) (*money.Money, error) {
	t.Helper()
	return money.New(amount, currency)
}

// Helper function to create a new Money instance from smallest unit for testing
func mustNewFromSmallestUnit(t *testing.T, amount int64, currency money.Code) *money.Money {
	t.Helper()
	m, err := money.NewFromSmallestUnit(amount, currency)
	require.NoError(t, err, "failed to create money from smallest unit for test")
	return m
}

func TestNewMoney_Precision(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		currency money.Code
		expected string
		wantErr  bool
	}{
		{"USD with cents", 100.50, money.USD, "100.50 USD", false},
		{"EUR with cents", 99.99, money.EUR, "99.99 EUR", false},
		{"JPY without cents", 1000.0, money.JPY, "1000 JPY", false},
		{"KWD with 3 decimals", 100.123, money.KWD, "100.12 KWD", false},
		{"Invalid currency", 100.50, money.Code("INVALID"), "", true},
		{"USD with more than 2 decimals", 100.999, money.USD, "101.00 USD", false},
		{"JPY with cents should round down", 1000.4, money.JPY, "1000 JPY", false},
		{"JPY with cents should round up", 1000.5, money.JPY, "1001 JPY", false},
		{"USD with exactly 2 decimals", 100.99, money.USD, "100.99 USD", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := newMoney(t, tt.amount, tt.currency)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if got, want := m.CurrencyCode(), tt.currency; got != want {
				t.Errorf("CurrencyCode() = %v, want %v", got, want)
			}
			assert.Equal(t, tt.expected, m.String())
		})
	}
}

func TestMoney_Arithmetic(t *testing.T) {
	usd100 := mustNew(t, 100.0, money.USD)
	usd50 := mustNew(t, 50.0, money.USD)
	eur100 := mustNew(t, 100.0, money.EUR)

	t.Run("Add same currency", func(t *testing.T) {
		result, err := usd100.Add(usd50)
		require.NoError(t, err)
		assert.InDelta(t, 150.0, result.AmountFloat(), 0.001)
		if got, want := result.CurrencyCode(), money.USD; got != want {
			t.Errorf("NewFromData() currency = %v, want %v", got, want)
		}
	})

	t.Run("Add different currency", func(t *testing.T) {
		_, err := usd100.Add(eur100)
		require.Error(t, err)
		assert.EqualError(t, err, "cannot add different currencies: USD and EUR")
	})

	t.Run("Subtract same currency", func(t *testing.T) {
		result, err := usd100.Subtract(usd50)
		require.NoError(t, err)
		assert.InDelta(t, 50.0, result.AmountFloat(), 0.001)
		if got, want := result.CurrencyCode(), money.USD; got != want {
			t.Errorf("Subtract() currency = %v, want %v", got, want)
		}
	})

	t.Run("Negate", func(t *testing.T) {
		result := usd100.Negate()
		assert.InDelta(t, -100.0, result.AmountFloat(), 0.001)
		if got, want := result.CurrencyCode(), money.USD; got != want {
			t.Errorf("Negate() currency = %v, want %v", got, want)
		}
	})

	t.Run("Add negative to money should subtract", func(t *testing.T) {
		usd1000 := mustNew(t, 1000.0, money.USD)
		result, err := usd1000.Add(usd100.Negate())
		require.NoError(t, err)
		assert.InDelta(t, 900, result.AmountFloat(), 0.01)
		if got, want := result.CurrencyCode(), money.USD; got != want {
			t.Errorf("Add() currency = %v, want %v", got, want)
		}
	})
}

func TestMoney_Comparison(t *testing.T) {
	usd100 := mustNew(t, 100.0, money.USD)
	usd50 := mustNew(t, 50.0, money.USD)
	eur100 := mustNew(t, 100.0, money.EUR)

	t.Run("Equals same currency", func(t *testing.T) {
		usd100b := mustNew(t, 100.0, money.USD)
		assert.True(t, usd100.Equals(usd100b))
		assert.False(t, usd100.Equals(usd50))
		if got, want := usd100b.CurrencyCode(), money.USD; got != want {
			t.Errorf("Equals() currency = %v, want %v", got, want)
		}
	})

	t.Run("Equals different currency", func(t *testing.T) {
		assert.False(t, usd100.Equals(eur100))
		if got, want := eur100.CurrencyCode(), money.EUR; got != want {
			t.Errorf("Equals() currency = %v, want %v", got, want)
		}
	})

	t.Run("GreaterThan same currency", func(t *testing.T) {
		result, err := usd100.GreaterThan(usd50)
		require.NoError(t, err)
		assert.True(t, result)
		if got, want := usd50.CurrencyCode(), money.USD; got != want {
			t.Errorf("GreaterThan() currency = %v, want %v", got, want)
		}

		result, err = usd50.GreaterThan(usd100)
		require.NoError(t, err)
		assert.False(t, result)
		if got, want := usd100.CurrencyCode(), money.USD; got != want {
			t.Errorf("GreaterThan() currency = %v, want %v", got, want)
		}
	})

	t.Run("GreaterThan different currency", func(t *testing.T) {
		_, err := usd100.GreaterThan(eur100)
		require.Error(t, err)
		assert.EqualError(t, err, "cannot compare different currencies: USD and EUR")
	})
}

func TestMoney_State(t *testing.T) {
	usd100 := mustNew(t, 100.0, money.USD)
	usd0 := mustNew(t, 0.0, money.USD)
	usdNeg50 := mustNew(t, -50.0, money.USD)

	t.Run("IsPositive", func(t *testing.T) {
		assert.True(t, usd100.IsPositive())
		assert.False(t, usd0.IsPositive())
		assert.False(t, usdNeg50.IsPositive())
		if got, want := usdNeg50.CurrencyCode(), money.USD; got != want {
			t.Errorf("IsPositive() currency = %v, want %v", got, want)
		}
	})

	t.Run("IsNegative", func(t *testing.T) {
		assert.False(t, usd100.IsNegative())
		assert.False(t, usd0.IsNegative())
		assert.True(t, usdNeg50.IsNegative())
		if got, want := usdNeg50.CurrencyCode(), money.USD; got != want {
			t.Errorf("IsNegative() currency = %v, want %v", got, want)
		}
	})

	t.Run("IsZero", func(t *testing.T) {
		assert.False(t, usd100.IsZero())
		assert.True(t, usd0.IsZero())
		assert.False(t, usdNeg50.IsZero())
		if got, want := usdNeg50.CurrencyCode(), money.USD; got != want {
			t.Errorf("IsZero() currency = %v, want %v", got, want)
		}
	})
}

func TestMoney_String(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		cc       money.Code
		expected string
	}{
		{"USD", 100.50, money.USD, "100.50 USD"},
		{"EUR", 99.99, money.EUR, "99.99 EUR"},
		{"JPY", 1000.0, money.JPY, "1000 JPY"},
		{"KWD", 100.123, money.KWD, "100.12 KWD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := mustNew(t, tt.amount, tt.cc)
			assert.Equal(t, tt.expected, m.String())
			if got, want := m.CurrencyCode(), tt.cc; got != want {
				t.Errorf("String() currency = %v, want %v", got, want)
			}
		})
	}
}

func TestMoney_PrecisionEdgeCases(t *testing.T) {
	t.Run("Very small amount", func(t *testing.T) {
		m := mustNew(t, 0.0001, money.USD)
		assert.InDelta(t, 0.00, m.AmountFloat(), 0.001)
	})

	t.Run("Very large amount", func(t *testing.T) {
		m := mustNew(t, 999999999999.99, money.USD)
		assert.InDelta(t, 999999999999.99, m.AmountFloat(), 0.001)
	})

	t.Run("Rounding down", func(t *testing.T) {
		m := mustNew(t, 100.994, money.USD)
		assert.InDelta(t, 100.99, m.AmountFloat(), 0.001)
	})

	t.Run("Rounding up", func(t *testing.T) {
		m := mustNew(t, 100.995, money.USD)
		assert.InDelta(t, 101.00, m.AmountFloat(), 0.001)
	})

	t.Run("JPY with decimals rounds up", func(t *testing.T) {
		m, err := money.New(1000.5, money.JPY)
		require.NoError(t, err)
		assert.InDelta(t, 1001.0, m.AmountFloat(), 0.001)
	})
}

func TestNewMoneyFromSmallestUnit(t *testing.T) {
	t.Run("USD cents", func(t *testing.T) {
		m := mustNewFromSmallestUnit(t, 1000, money.USD)
		assert.Equal(t, int64(1000), m.Amount())
		if got, want := m.CurrencyCode(), money.USD; got != want {
			t.Errorf("NewFromSmallestUnit() currency = %v, want %v", got, want)
		}
		assert.InDelta(t, 10.0, m.AmountFloat(), 0.001)
	})

	t.Run("JPY yen", func(t *testing.T) {
		m := mustNewFromSmallestUnit(t, 1000, money.JPY)
		assert.Equal(t, int64(1000), m.Amount())
		if got, want := m.CurrencyCode(), money.JPY; got != want {
			t.Errorf("NewFromSmallestUnit() currency = %v, want %v", got, want)
		}
		assert.InDelta(t, 1000.0, m.AmountFloat(), 0.001)
	})

	t.Run("Invalid currency", func(t *testing.T) {
		_, err := money.NewFromSmallestUnit(100, money.Code("INVALID"))
		require.Error(t, err)
		assert.ErrorIs(t, err, money.ErrInvalidCurrency)
	})
}

func TestMoney_Abs(t *testing.T) {
	t.Run("Positive amount", func(t *testing.T) {
		m := mustNew(t, 100.0, money.USD)
		result := m.Abs()
		assert.Equal(t, m.Amount(), result.Amount())
		if got, want := result.CurrencyCode(), money.USD; got != want {
			t.Errorf("Abs() currency = %v, want %v", got, want)
		}
	})

	t.Run("Negative amount", func(t *testing.T) {
		m := mustNew(t, -100.0, money.USD)
		result := m.Abs()
		assert.Equal(t, int64(10000), result.Amount())
		if got, want := result.CurrencyCode(), money.USD; got != want {
			t.Errorf("Abs() currency = %v, want %v", got, want)
		}
	})

	t.Run("Zero amount", func(t *testing.T) {
		m := mustNew(t, 0.0, money.USD)
		result := m.Abs()
		assert.Equal(t, m.Amount(), result.Amount())
		if got, want := result.CurrencyCode(), money.USD; got != want {
			t.Errorf("Abs() currency = %v, want %v", got, want)
		}
	})
}

func TestMoney_Multiply(t *testing.T) {
	tests := []struct {
		name    string
		amount  float64
		factor  float64
		expect  float64
		wantErr bool
	}{
		{"Multiply by 1", 100.50, 1.0, 100.50, false},
		{"Multiply by 2", 100.50, 2.0, 201.00, false},
		{"Multiply by 0.5", 100.50, 0.5, 50.25, false},
		{"Multiply by 0", 100.50, 0.0, 0.0, false},
		{"Multiply by negative", 100.50, -1.0, -100.50, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := mustNew(t, tc.amount, money.USD)

			result, err := m.Multiply(tc.factor)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tc.expect == 0.0 {
					// Special case for zero - use direct equality
					assert.InDelta(t, 0.0, result.AmountFloat(), 0.001)
				} else {
					assert.InEpsilon(
						t,
						tc.expect,
						result.AmountFloat(),
						0.01,
						"Expected %v to be within epsilon of %v",
						result.AmountFloat(),
						tc.expect,
					)
				}
				assert.Equal(t, money.USD, result.CurrencyCode())
			}
		})
	}
}

func TestMoney_Divide(t *testing.T) {
	m := mustNew(t, 100.0, money.USD)

	t.Run("Divide by 2", func(t *testing.T) {
		result, err := m.Divide(2.0)
		require.NoError(t, err)
		assert.InDelta(t, 50.0, result.AmountFloat(), 0.001)
		if got, want := result.CurrencyCode(), money.USD; got != want {
			t.Errorf("Divide() currency = %v, want %v", got, want)
		}
	})

	t.Run("Divide by 4", func(t *testing.T) {
		result, err := m.Divide(4.0)
		require.NoError(t, err)
		assert.InDelta(t, 25.0, result.AmountFloat(), 0.001)
		if got, want := result.CurrencyCode(), money.USD; got != want {
			t.Errorf("Divide() currency = %v, want %v", got, want)
		}
	})

	t.Run("Divide by 0", func(t *testing.T) {
		_, err := m.Divide(0.0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "divisor must be positive")
	})

	t.Run("Divide by 3 (precision loss)", func(t *testing.T) {
		result, err := m.Divide(3.0)
		require.NoError(t, err)
		// Check that the result is approximately correct
		assert.InDelta(t, 33.33, result.AmountFloat(), 0.01)
		if got, want := result.CurrencyCode(), money.USD; got != want {
			t.Errorf("Divide() currency = %v, want %v", got, want)
		}
	})
}

func TestMoney_JPY(t *testing.T) {
	t.Run("JPY whole number valid", func(t *testing.T) {
		m := mustNew(t, 1000, money.JPY)
		assert.Equal(t, int64(1000), m.Amount())
		assert.Equal(t, money.JPY, m.CurrencyCode())
		assert.InDelta(t, 1000.0, m.AmountFloat(), 0.001)
	})

	t.Run("JPY with .0 valid", func(t *testing.T) {
		m := mustNew(t, 5000.0, money.JPY)
		assert.Equal(t, int64(5000), m.Amount())
		assert.Equal(t, money.JPY, m.CurrencyCode())
		assert.InDelta(t, 5000.0, m.AmountFloat(), 0.001)
	})

	t.Run("JPY with decimals rounds up", func(t *testing.T) {
		m, err := money.New(1000.5, money.JPY)
		require.NoError(t, err)
		assert.Equal(t, int64(1001), m.Amount())
		assert.Equal(t, money.JPY, m.CurrencyCode())
		assert.InDelta(t, 1001.0, m.AmountFloat(), 0.001)
	})

	t.Run("JPY with two decimals rounds", func(t *testing.T) {
		m, err := money.New(1234.56, money.JPY)
		require.NoError(t, err)
		assert.Equal(t, int64(1235), m.Amount())
		assert.Equal(t, money.JPY, m.CurrencyCode())
		assert.InDelta(t, 1235.0, m.AmountFloat(), 0.001)
	})

	t.Run("JPY negative whole number valid", func(t *testing.T) {
		m, err := money.New(-2000, money.JPY)
		require.NoError(t, err)
		assert.Equal(t, int64(-2000), m.Amount())
		assert.Equal(t, money.JPY, m.CurrencyCode())
		assert.InDelta(t, -2000.0, m.AmountFloat(), 0.001)
	})

	t.Run("JPY zero valid", func(t *testing.T) {
		m, err := money.New(0, money.JPY)
		require.NoError(t, err)
		assert.Equal(t, int64(0), m.Amount())
		assert.Equal(t, money.JPY, m.CurrencyCode())
		assert.InDelta(t, 0.0, m.AmountFloat(), 0.001)
	})
}

// TestMoney_Add is covered by TestMoney_Arithmetic
// TestMoney_Subtract is covered by TestMoney_Arithmetic
// TestMoney_Multiply is covered by the comprehensive test above
// TestMoney_Divide is covered by the comprehensive test above
// TestMoney_Compare is covered by TestMoney_Comparison
// TestMoney_IsZero is covered by TestMoney_State
