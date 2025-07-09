package money_test

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMoney_Precision(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		currency currency.Code
		wantErr  bool
	}{
		{"USD with cents", 100.50, "USD", false},
		{"EUR with cents", 99.99, "EUR", false},
		{"JPY whole numbers", 1000.0, "JPY", false},
		{"KWD with 3 decimals", 100.123, "KWD", false},
		{"Too many decimals for USD", 100.123, "USD", true},
		{"Too many decimals for JPY", 100.5, "JPY", true},
		{"Invalid currency", 100.0, "INVALID", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			money, err := money.NewMoney(tt.amount, tt.currency)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.currency, money.Currency())
			assert.InDelta(t, tt.amount, money.AmountFloat(), 0.001)
		})
	}
}

func TestMoney_Arithmetic(t *testing.T) {
	usd100, err := money.NewMoney(100.0, "USD")
	require.NoError(t, err)
	usd50, err := money.NewMoney(50.0, "USD")
	require.NoError(t, err)
	eur100, err := money.NewMoney(100.0, "EUR")
	require.NoError(t, err)

	t.Run("Add same currency", func(t *testing.T) {
		result, err := usd100.Add(usd50)
		require.NoError(t, err)
		assert.InDelta(t, 150.0, result.AmountFloat(), 0.001)
		assert.Equal(t, "USD", string(result.Currency()))
	})

	t.Run("Add different currency", func(t *testing.T) {
		_, err := usd100.Add(eur100)
		assert.Error(t, err)
		assert.ErrorIs(t, err, common.ErrInvalidCurrencyCode)
	})

	t.Run("Subtract same currency", func(t *testing.T) {
		result, err := usd100.Subtract(usd50)
		require.NoError(t, err)
		assert.InDelta(t, 50.0, result.AmountFloat(), 0.001)
		assert.Equal(t, "USD", string(result.Currency()))
	})

	t.Run("Negate", func(t *testing.T) {
		result := usd100.Negate()
		assert.InDelta(t, -100.0, result.AmountFloat(), 0.001)
		assert.Equal(t, "USD", string(result.Currency()))
	})
}

func TestMoney_Comparison(t *testing.T) {
	usd100, err := money.NewMoney(100.0, "USD")
	require.NoError(t, err)
	usd50, err := money.NewMoney(50.0, "USD")
	require.NoError(t, err)
	eur100, err := money.NewMoney(100.0, "EUR")
	require.NoError(t, err)

	t.Run("Equals same currency", func(t *testing.T) {
		usd100b, err := money.NewMoney(100.0, "USD")
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

	t.Run("GreaterThan different currency", func(t *testing.T) {
		_, err := usd100.GreaterThan(eur100)
		assert.Error(t, err)
		assert.ErrorIs(t, err, common.ErrInvalidCurrencyCode)
	})
}

func TestMoney_State(t *testing.T) {
	usd100, err := money.NewMoney(100.0, "USD")
	require.NoError(t, err)
	usd0, err := money.NewMoney(0.0, "USD")
	require.NoError(t, err)
	usdNeg50, err := money.NewMoney(-50.0, "USD")
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
		cc       currency.Code
		expected string
	}{
		{"USD", 100.50, "USD", "100.50 USD"},
		{"EUR", 99.99, "EUR", "99.99 EUR"},
		{"JPY", 1000.0, "JPY", "1000 JPY"},
		{"KWD", 100.123, "KWD", "100.123 KWD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			money, err := money.NewMoney(tt.amount, tt.cc)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, money.String())
		})
	}
}

func TestMoney_PrecisionEdgeCases(t *testing.T) {
	t.Run("USD with exactly 2 decimals", func(t *testing.T) {
		money, err := money.NewMoney(100.99, "USD")
		require.NoError(t, err)
		assert.InDelta(t, 100.99, money.AmountFloat(), 0.001)
	})

	t.Run("JPY with no decimals", func(t *testing.T) {
		money, err := money.NewMoney(1000.0, "JPY")
		require.NoError(t, err)
		assert.InDelta(t, 1000.0, money.AmountFloat(), 0.001)
	})

	t.Run("KWD with exactly 3 decimals", func(t *testing.T) {
		money, err := money.NewMoney(100.123, "KWD")
		require.NoError(t, err)
		assert.InDelta(t, 100.123, money.AmountFloat(), 0.001)
	})
}

func TestNewMoneyFromSmallestUnit(t *testing.T) {
	t.Run("USD from cents", func(t *testing.T) {
		m, err := money.NewMoneyFromSmallestUnit(10050, "USD") // 100.50 USD
		require.NoError(t, err)
		assert.Equal(t, int64(10050), m.Amount())
		assert.Equal(t, "USD", string(m.Currency()))
		assert.InDelta(t, 100.50, m.AmountFloat(), 0.001)
	})

	t.Run("JPY from yen", func(t *testing.T) {
		money, err := money.NewMoneyFromSmallestUnit(1000, "JPY") // 1000 JPY
		require.NoError(t, err)
		assert.Equal(t, int64(1000), money.Amount())
		assert.Equal(t, "JPY", string(money.Currency()))
		assert.InDelta(t, 1000.0, money.AmountFloat(), 0.001)
	})

	t.Run("Invalid currency", func(t *testing.T) {
		_, err := money.NewMoneyFromSmallestUnit(100, "INVALID")
		assert.Error(t, err)
		assert.ErrorIs(t, err, common.ErrInvalidCurrencyCode)
	})
}

func TestMoney_Abs(t *testing.T) {
	t.Run("Positive amount", func(t *testing.T) {
		money, err := money.NewMoney(100.0, "USD")
		require.NoError(t, err)
		result := money.Abs()
		assert.Equal(t, money.Amount(), result.Amount())
	})

	t.Run("Negative amount", func(t *testing.T) {
		money, err := money.NewMoney(-100.0, "USD")
		require.NoError(t, err)
		result := money.Abs()
		assert.Equal(t, int64(10000), result.Amount()) // 100.00 USD in cents
	})

	t.Run("Zero amount", func(t *testing.T) {
		money, err := money.NewMoney(0.0, "USD")
		require.NoError(t, err)
		result := money.Abs()
		assert.Equal(t, money.Amount(), result.Amount())
	})
}

func TestMoney_Multiply(t *testing.T) {
	money, err := money.NewMoney(100.0, "USD")
	require.NoError(t, err)

	t.Run("Multiply by 2", func(t *testing.T) {
		result, err := money.Multiply(2.0)
		require.NoError(t, err)
		assert.InDelta(t, 200.0, result.AmountFloat(), 0.001)
		assert.Equal(t, "USD", string(result.Currency()))
	})

	t.Run("Multiply by 0.5", func(t *testing.T) {
		result, err := money.Multiply(0.5)
		require.NoError(t, err)
		assert.InDelta(t, 50.0, result.AmountFloat(), 0.001)
		assert.Equal(t, "USD", string(result.Currency()))
	})

	t.Run("Multiply by 0", func(t *testing.T) {
		result, err := money.Multiply(0.0)
		require.NoError(t, err)
		assert.InDelta(t, 0.0, result.AmountFloat(), 0.001)
		assert.Equal(t, "USD", string(result.Currency()))
	})

	t.Run("Multiply by negative", func(t *testing.T) {
		result, err := money.Multiply(-1.0)
		require.NoError(t, err)
		assert.InDelta(t, -100.0, result.AmountFloat(), 0.001)
		assert.Equal(t, "USD", string(result.Currency()))
	})
}

func TestMoney_Divide(t *testing.T) {
	money, err := money.NewMoney(100.0, "USD")
	require.NoError(t, err)

	t.Run("Divide by 2", func(t *testing.T) {
		result, err := money.Divide(2.0)
		require.NoError(t, err)
		assert.InDelta(t, 50.0, result.AmountFloat(), 0.001)
		assert.Equal(t, "USD", string(result.Currency()))
	})

	t.Run("Divide by 4", func(t *testing.T) {
		result, err := money.Divide(4.0)
		require.NoError(t, err)
		assert.InDelta(t, 25.0, result.AmountFloat(), 0.001)
		assert.Equal(t, "USD", string(result.Currency()))
	})

	t.Run("Divide by 0", func(t *testing.T) {
		_, err := money.Divide(0.0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "division by zero")
	})

	t.Run("Divide by 3 (precision loss)", func(t *testing.T) {
		_, err := money.Divide(3.0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "precision loss")
	})
}
