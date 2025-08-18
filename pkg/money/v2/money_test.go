package money_test

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/money"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	usd = money.Currency{Code: "USD", Decimals: 2}
	jpy = money.Currency{Code: "JPY", Decimals: 0}
)

func TestNewMoney_Precision(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		currency money.Currency
		expected float64
		wantErr  bool
	}{
		{"USD with cents", 100.50, usd, 100.50, false},
		{"EUR with cents", 99.99, money.Currency{Code: "EUR", Decimals: 2}, 99.99, false},
		{"JPY whole numbers", 1000.0, jpy, 1000.0, false},
		{"KWD with 3 decimals", 100.123, money.Currency{Code: "KWD", Decimals: 3}, 100.123, false},
		{"Too many decimals for USD", 100.123, usd, 100.12, false}, // Rounds to 2 decimals
		{
			"Too many decimals for JPY",
			1000.5,
			jpy,
			1001.0,
			false, // Rounds up to nearest whole number
		},
		{"Invalid currency", 100.0, money.Currency{Code: "INVALID", Decimals: 2}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			money, err := money.New(tt.amount, tt.currency)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.currency, money.Currency())
			assert.InDelta(t, tt.expected, money.AmountFloat(), 0.001)
		})
	}
}

func TestMoney_Arithmetic(t *testing.T) {
	usd100, err := money.New(100.0, usd)
	require.NoError(t, err)
	usd50, err := money.New(50.0, usd)
	require.NoError(t, err)
	eur100, err := money.New(100.0, money.Currency{Code: "EUR", Decimals: 2})
	require.NoError(t, err)

	t.Run("Add same currency", func(t *testing.T) {
		result, err := usd100.Add(usd50)
		require.NoError(t, err)
		assert.InDelta(t, 150.0, result.AmountFloat(), 0.001)
		assert.Equal(t, usd, result.Currency())
	})

	t.Run("Add different currency", func(t *testing.T) {
		_, err := usd100.Add(eur100)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot add different currencies")
	})

	t.Run("Subtract same currency", func(t *testing.T) {
		result, err := usd100.Subtract(usd50)
		require.NoError(t, err)
		assert.InDelta(t, 50.0, result.AmountFloat(), 0.001)
		assert.Equal(t, usd, result.Currency())
	})

	t.Run("Subtract different currency", func(t *testing.T) {
		_, err := usd100.Subtract(eur100)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot subtract different currencies")
	})

	t.Run("Negate", func(t *testing.T) {
		result := usd100.Negate()
		assert.InDelta(t, -100.0, result.AmountFloat(), 0.001)
		assert.Equal(t, usd, result.Currency())
	})

	t.Run("Add negative to money should subtract", func(t *testing.T) {
		usd1000, err := money.New(1000.0, usd)
		require.NoError(t, err)
		result, err := usd1000.Add(usd100.Negate())
		require.NoError(t, err)
		assert.InDelta(t, 900, result.AmountFloat(), 0.01)
	})
}

func TestMoney_Comparison(t *testing.T) {
	usd100, err := money.New(100.0, usd)
	require.NoError(t, err)
	usd50, err := money.New(50.0, usd)
	require.NoError(t, err)
	eur100, err := money.New(100.0, money.Currency{Code: "EUR", Decimals: 2})
	require.NoError(t, err)

	t.Run("Equals same currency", func(t *testing.T) {
		usd100b, err := money.New(100.0, usd)
		require.NoError(t, err)
		assert.True(t, usd100.Equals(usd100b))
		assert.False(t, usd100.Equals(usd50))
	})

	t.Run("Equals different currency", func(t *testing.T) {
		assert.False(t, usd100.Equals(eur100))
	})

	t.Run("GreaterThan same currency", func(t *testing.T) {
		result, err := usd100.GreaterThan(usd50)
		require.NoError(t, err)
		assert.True(t, result)

		result, err = usd50.GreaterThan(usd100)
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("LessThan same currency", func(t *testing.T) {
		result, err := usd50.LessThan(usd100)
		require.NoError(t, err)
		assert.True(t, result)

		result, err = usd100.LessThan(usd50)
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("LessThan different currency", func(t *testing.T) {
		_, err := usd100.LessThan(eur100)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid currency code")
	})
}

func TestMoney_State(t *testing.T) {
	usd100, err := money.New(100.0, "USD")
	require.NoError(t, err)
	usd0, err := money.New(0.0, "USD")
	require.NoError(t, err)
	usdNeg50, err := money.New(-50.0, "USD")
	require.NoError(t, err)

	t.Run("IsPositive", func(t *testing.T) {
		assert.True(t, usd100.IsPositive())
		assert.False(t, usd0.IsPositive())
		assert.False(t, usdNeg50.IsPositive())
	})

	t.Run("IsNegative", func(t *testing.T) {
		assert.False(t, usd100.IsNegative())
		assert.False(t, usd0.IsNegative())
		assert.True(t, usdNeg50.IsNegative())
	})

	t.Run("IsZero", func(t *testing.T) {
		assert.False(t, usd100.IsZero())
		assert.True(t, usd0.IsZero())
		assert.False(t, usdNeg50.IsZero())
	})
}

func TestMoney_String(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		cc       money.Currency
		expected string
	}{
		{"USD", 100.50, usd, "100.50 USD"},
		{"EUR", 99.99, money.Currency{Code: "EUR", Decimals: 2}, "99.99 EUR"},
		{"JPY", 1000.0, jpy, "1000 JPY"},
		{"KWD", 100.123, money.Currency{Code: "KWD", Decimals: 3}, "100.123 KWD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			money, err := money.New(tt.amount, tt.cc)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, money.String())
		})
	}
}

func TestMoney_PrecisionEdgeCases(t *testing.T) {
	t.Run("USD with exactly 2 decimals", func(t *testing.T) {
		money, err := money.New(100.99, usd)
		require.NoError(t, err)
		assert.InDelta(t, 100.99, money.AmountFloat(), 0.001)
	})

	t.Run("JPY with no decimals", func(t *testing.T) {
		money, err := money.New(1000.0, jpy)
		require.NoError(t, err)
		assert.InDelta(t, 1000.0, money.AmountFloat(), 0.001)
	})

	t.Run("KWD with exactly 3 decimals", func(t *testing.T) {
		kwd := money.Currency{Code: "KWD", Decimals: 3}
		money, err := money.New(100.123, kwd)
		require.NoError(t, err)
		assert.InDelta(t, 100.123, money.AmountFloat(), 0.001)
	})
}

func TestNewMoneyFromSmallestUnit(t *testing.T) {
	t.Run("USD from cents", func(t *testing.T) {
		m, err := money.NewFromSmallestUnit(10050, usd)
		require.NoError(t, err)
		assert.Equal(t, int64(10050), m.Amount())
		assert.Equal(t, usd, m.Currency())
		assert.InDelta(t, 100.50, m.AmountFloat(), 0.001)
	})

	t.Run("JPY from yen", func(t *testing.T) {
		m, err := money.NewFromSmallestUnit(1000, jpy)
		require.NoError(t, err)
		assert.Equal(t, int64(1000), m.Amount())
		assert.Equal(t, jpy, m.Currency())
		assert.InDelta(t, 1000.0, m.AmountFloat(), 0.001)
	})

	t.Run("Invalid currency", func(t *testing.T) {
		invalidCurrency := money.Currency{Code: "INVALID", Decimals: 2}
		_, err := money.NewFromSmallestUnit(100, invalidCurrency)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid currency")
	})
}

func TestMoney_Abs(t *testing.T) {
	t.Run("Positive amount", func(t *testing.T) {
		money, err := money.New(100.0, usd)
		require.NoError(t, err)
		result := money.Abs()
		assert.Equal(t, money.Amount(), result.Amount())
	})

	t.Run("Negative amount", func(t *testing.T) {
		money, err := money.New(-100.0, "USD")
		require.NoError(t, err)
		result := money.Abs()
		assert.Equal(t, int64(10000), result.Amount()) // 100.00 USD in cents
	})

	t.Run("Zero amount", func(t *testing.T) {
		money, err := money.New(0.0, usd)
		require.NoError(t, err)
		result := money.Abs()
		assert.Equal(t, money.Amount(), result.Amount())
	})
}

func TestMoney_Multiply(t *testing.T) {
	money, err := money.New(100.0, usd)
	require.NoError(t, err)

	t.Run("Multiply by negative", func(t *testing.T) {
		_, err := money.Multiply(-1.0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "factor cannot be negative")
	})

	t.Run("Multiply by 2", func(t *testing.T) {
		result, err := money.Multiply(2.0)
		require.NoError(t, err)
		assert.InDelta(t, 200.0, result.AmountFloat(), 0.001)
		assert.Equal(t, usd, result.Currency())
	})

	t.Run("Multiply by 0.5", func(t *testing.T) {
		result, err := money.Multiply(0.5)
		require.NoError(t, err)
		assert.InDelta(t, 50.0, result.AmountFloat(), 0.001)
		assert.Equal(t, usd, result.Currency())
	})

	t.Run("Multiply by 0", func(t *testing.T) {
		result, err := money.Multiply(0.0)
		require.NoError(t, err)
		assert.InDelta(t, 0.0, result.AmountFloat(), 0.001)
		assert.Equal(t, usd, result.Currency())
	})

	t.Run("Multiply by negative", func(t *testing.T) {
		_, err := money.Multiply(-1.0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "factor cannot be negative")
	})
}

func TestMoney_Divide(t *testing.T) {
	money, err := money.New(100.0, usd)
	require.NoError(t, err)

	t.Run("Divide by 3 (precision loss)", func(t *testing.T) {
		// Should round to 2 decimal places for USD
		result, err := money.Divide(3.0)
		require.NoError(t, err)
		assert.InDelta(t, 33.33, result.AmountFloat(), 0.01)
	})

	t.Run("Divide by 2", func(t *testing.T) {
		result, err := money.Divide(2.0)
		require.NoError(t, err)
		assert.InDelta(t, 50.0, result.AmountFloat(), 0.001)
		assert.Equal(t, usd, result.Currency())
	})

	t.Run("Divide by 4", func(t *testing.T) {
		result, err := money.Divide(4.0)
		require.NoError(t, err)
		assert.InDelta(t, 25.0, result.AmountFloat(), 0.001)
		assert.Equal(t, usd, result.Currency())
	})

	t.Run("Divide by 0", func(t *testing.T) {
		_, err := money.Divide(0.0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "divisor must be positive")
	})

	t.Run("Divide by 3 (precision loss)", func(t *testing.T) {
		// Should round to 2 decimal places for USD
		result, err := money.Divide(3.0)
		require.NoError(t, err)
		assert.InDelta(t, 33.33, result.AmountFloat(), 0.01)
	})
}

func TestNewMoney_JPY(t *testing.T) {
	t.Run("JPY whole number valid", func(t *testing.T) {
		m, err := money.New(1000, "JPY")
		require.NoError(t, err)
		assert.Equal(t, int64(1000), m.Amount())
		assert.Equal(t, money.Currency{Code: "JPY", Decimals: 0}, m.Currency())
		assert.InDelta(t, 1000.0, m.AmountFloat(), 0.001)
	})

	t.Run("JPY with .0 valid", func(t *testing.T) {
		m, err := money.New(5000.0, "JPY")
		require.NoError(t, err)
		assert.Equal(t, int64(5000), m.Amount())
		assert.Equal(t, money.Currency{Code: "JPY", Decimals: 0}, m.Currency())
		assert.InDelta(t, 5000.0, m.AmountFloat(), 0.001)
	})

	t.Run("JPY with decimals rounds up", func(t *testing.T) {
		m, err := money.New(1000.5, "JPY")
		require.NoError(t, err)
		assert.Equal(t, int64(1001), m.Amount())
		assert.Equal(t, money.Currency{Code: "JPY", Decimals: 0}, m.Currency())
		assert.InDelta(t, 1001.0, m.AmountFloat(), 0.001)
	})

	t.Run("JPY with two decimals rounds", func(t *testing.T) {
		m, err := money.New(1234.56, "JPY")
		require.NoError(t, err)
		assert.Equal(t, int64(1235), m.Amount())
		assert.Equal(t, money.Currency{Code: "JPY", Decimals: 0}, m.Currency())
		assert.InDelta(t, 1235.0, m.AmountFloat(), 0.001)
	})

	t.Run("JPY negative whole number valid", func(t *testing.T) {
		m, err := money.New(-2000, "JPY")
		require.NoError(t, err)
		assert.Equal(t, int64(-2000), m.Amount())
		assert.Equal(t, money.Currency{Code: "JPY", Decimals: 0}, m.Currency())
		assert.InDelta(t, -2000.0, m.AmountFloat(), 0.001)
	})

	t.Run("JPY zero valid", func(t *testing.T) {
		m, err := money.New(0, "JPY")
		require.NoError(t, err)
		assert.Equal(t, int64(0), m.Amount())
		assert.Equal(t, money.Currency{Code: "JPY", Decimals: 0}, m.Currency())
		assert.InDelta(t, 0.0, m.AmountFloat(), 0.001)
	})
}
