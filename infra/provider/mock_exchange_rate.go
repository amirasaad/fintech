package provider

import (
	"context"
	"time"

	"github.com/amirasaad/fintech/pkg/provider"
)

// mockExchangeRate is a mock implementation of the ExchangeRate interface for testing.
type mockExchangeRate struct {
	GetRateFunc  func(ctx context.Context, from, to string) (*provider.ExchangeInfo, error)
	GetRatesFunc func(
		ctx context.Context,
		from string,
	) (map[string]*provider.ExchangeInfo, error)
	IsSupportedFunc func(from, to string) bool
	NameFunc        func() string
	IsHealthyFunc   func() bool
}

func NewMockExchangeRate() *mockExchangeRate {
	return &mockExchangeRate{
		GetRateFunc: func(ctx context.Context, from, to string) (*provider.ExchangeInfo, error) {
			return &provider.ExchangeInfo{
				OriginalCurrency:  from,
				ConvertedCurrency: to,
				ConversionRate:    1.0,
				Timestamp:         time.Now(),
			}, nil
		},
		GetRatesFunc: func(
			ctx context.Context,
			from string,
		) (map[string]*provider.ExchangeInfo, error) {
			result := make(map[string]*provider.ExchangeInfo)
			supportedCurrencies := []string{
				"EUR", "GBP", "JPY", "USD", "AUD",
				"CAD", "CHF", "CNY", "HKD", "NZD",
				"SGD", "ZAR",
			}
			for _, currency := range supportedCurrencies {
				result[currency] = &provider.ExchangeInfo{
					OriginalCurrency:  from,
					ConvertedCurrency: currency,
					ConversionRate:    1.0,
					Timestamp:         time.Now(),
				}
			}
			return result, nil
		},
		IsSupportedFunc: func(from, to string) bool {
			return true
		},
		NameFunc: func() string {
			return "mock"
		},
		IsHealthyFunc: func() bool {
			return true
		},
	}
}

// GetRate calls the mock implementation of GetRate.
func (m *mockExchangeRate) GetRate(
	ctx context.Context,
	from, to string,
) (*provider.ExchangeInfo, error) {
	if m.GetRateFunc != nil {
		return m.GetRateFunc(ctx, from, to)
	}
	return &provider.ExchangeInfo{
		OriginalCurrency:  from,
		ConvertedCurrency: to,
		ConversionRate:    1.0,
		Timestamp:         time.Now(),
	}, nil
}

// GetRates calls the mock implementation of GetRates.
func (m *mockExchangeRate) GetRates(
	ctx context.Context,
	from string,
) (map[string]*provider.ExchangeInfo, error) {
	if m.GetRatesFunc != nil {
		return m.GetRatesFunc(ctx, from)
	}
	result := make(map[string]*provider.ExchangeInfo)
	supportedCurrencies := []string{
		"EUR", "GBP", "JPY", "USD", "AUD",
		"CAD", "CHF", "CNY", "HKD", "NZD",
		"SGD", "ZAR",
	}
	for _, currency := range supportedCurrencies {
		result[currency] = &provider.ExchangeInfo{
			OriginalCurrency:  from,
			ConvertedCurrency: currency,
			ConversionRate:    1.0,
			Timestamp:         time.Now(),
		}
	}
	return result, nil
}

// IsSupported calls the mock implementation of IsSupported.
func (m *mockExchangeRate) IsSupported(from, to string) bool {
	if m.IsSupportedFunc != nil {
		return m.IsSupportedFunc(from, to)
	}
	return true
}

// Name calls the mock implementation of Name.
func (m *mockExchangeRate) Name() string {
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return "mock"
}

// IsHealthy calls the mock implementation of IsHealthy.
func (m *mockExchangeRate) IsHealthy() bool {
	if m.IsHealthyFunc != nil {
		return m.IsHealthyFunc()
	}
	return true
}
