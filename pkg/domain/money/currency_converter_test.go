package money_test

import (
	"errors"
	"math"
	"testing"

	"github.com/amirasaad/fintech/infra/provider"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amirasaad/fintech/internal/fixtures"
)

func TestCurrencyConversion(t *testing.T) {
	tests := []struct {
		name    string
		amount  float64
		from    string
		to      string
		want    *domain.ConversionInfo
		wantErr bool
	}{
		{"USD to EUR", 100, "USD", "EUR", &domain.ConversionInfo{
			OriginalAmount:    100,
			OriginalCurrency:  "USD",
			ConvertedAmount:   90,
			ConvertedCurrency: "EUR",
			ConversionRate:    0.9,
		}, false},
		{"EUR to GBP", 100, "EUR", "GBP", &domain.ConversionInfo{
			OriginalAmount:    100,
			OriginalCurrency:  "EUR",
			ConvertedAmount:   80,
			ConvertedCurrency: "GBP",
			ConversionRate:    0.8,
		}, false},
		{"Unsupported currency", 100, "USD", "XXX", nil, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			converter := fixtures.NewMockCurrencyConverter(t)
			var err error
			if test.wantErr {
				err = errors.New("test error")
			}

			converter.EXPECT().Convert(test.amount, test.from, test.to).Return(test.want, err)
			converted, err := converter.Convert(test.amount, test.from, test.to)
			if (err != nil) != test.wantErr {
				t.Errorf("Convert() error = %v, wantErr %v", err, test.wantErr)
			}
			if converted != nil {
				if converted.ConvertedAmount != test.want.ConvertedAmount {
					t.Errorf("Convert() = %v, want %v", converted, test.want)
				}
			}
		})
	}
}

func TestJPYToUSDConversion(t *testing.T) {
	converter := provider.NewStubCurrencyConverter()

	t.Run("Large JPY to USD conversion, no decimals", func(t *testing.T) {
		amountJPY := 1_000_000_000.0 // 1 billion JPY
		convInfo, err := converter.Convert(amountJPY, "JPY", "USD")
		require.NoError(t, err)
		usd, err := money.NewMoney(convInfo.ConvertedAmount, "USD")
		require.NoError(t, err)
		assert.Equal(t, "USD", string(usd.Currency()))
		assert.InDelta(t, convInfo.ConvertedAmount, usd.AmountFloat(), 0.01)
	})

	t.Run("JPY to USD conversion with float imprecision", func(t *testing.T) {
		amountJPY := 6832299.83 // value that could cause float imprecision
		convInfo, err := converter.Convert(amountJPY, "JPY", "USD")
		require.NoError(t, err)
		usd, err := money.NewMoney(convInfo.ConvertedAmount, "USD")
		require.NoError(t, err)
		assert.Equal(t, "USD", string(usd.Currency()))
		// Should be rounded to 2 decimals
		meta, _ := currency.Get("USD")
		factor := math.Pow10(meta.Decimals)
		expected := math.Round(convInfo.ConvertedAmount*factor) / factor
		assert.InDelta(t, expected, usd.AmountFloat(), 0.001)
	})

	t.Run("JPY to USD conversion with too many decimals for USD", func(t *testing.T) {
		amountJPY := 1234567.89
		convInfo, err := converter.Convert(amountJPY, "JPY", "USD")
		require.NoError(t, err)
		// Manually add extra decimals to simulate imprecision
		usdAmount := convInfo.ConvertedAmount + 0.001234
		usd, err := money.NewMoney(usdAmount, "USD")
		require.NoError(t, err)
		meta, _ := currency.Get("USD")
		factor := math.Pow10(meta.Decimals)
		expected := math.Round(usdAmount*factor) / factor
		assert.InDelta(t, expected, usd.AmountFloat(), 0.001)
	})

	t.Run("JPY to USD conversion with zero", func(t *testing.T) {
		convInfo, err := converter.Convert(0, "JPY", "USD")
		require.NoError(t, err)
		usd, err := money.NewMoney(convInfo.ConvertedAmount, "USD")
		require.NoError(t, err)
		assert.Equal(t, int64(0), usd.Amount())
	})
}
