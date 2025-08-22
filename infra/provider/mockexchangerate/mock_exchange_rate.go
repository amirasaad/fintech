package mockexchangerate

import (
	"context"
	"time"

	"github.com/amirasaad/fintech/pkg/provider/exchange"
)

// mockExchangeRate is a mock implementation of the ExchangeRate interface for testing.
type mockExchangeRate struct {
	GetRateFunc  func(ctx context.Context, from, to string) (*exchange.RateInfo, error)
	GetRatesFunc func(
		ctx context.Context,
		from string,
		to []string,
	) (map[string]*exchange.RateInfo, error)
	IsSupportedFunc func(from, to string) bool
	NameFunc        func() string
	IsHealthyFunc   func(ctx context.Context) error
}

func NewMockExchangeRate() *mockExchangeRate {
	return &mockExchangeRate{
		GetRateFunc: func(ctx context.Context, from, to string) (*exchange.RateInfo, error) {
			return &exchange.RateInfo{
				FromCurrency: from,
				ToCurrency:   to,
				Rate:         1.0,
				Timestamp:    time.Now(),
			}, nil
		},
		GetRatesFunc: func(
			ctx context.Context,
			from string,
			to []string,
		) (map[string]*exchange.RateInfo, error) {
			result := make(map[string]*exchange.RateInfo)
			allSupportedCurrencies := []string{
				"EUR", "GBP", "JPY", "USD", "AUD",
				"CAD", "CHF", "CNY", "HKD", "NZD",
				"SGD", "ZAR",
			}

			// If 'to' is empty, return all supported currencies
			if len(to) == 0 {
				to = allSupportedCurrencies
			}

			for _, currency := range to {
				// Check if the currency is in our list of all supported currencies
				found := false
				for _, supported := range allSupportedCurrencies {
					if currency == supported {
						found = true
						break
					}
				}
				if !found {
					continue // Skip if not a supported currency
				}

				result[currency] = &exchange.RateInfo{
					FromCurrency: from,
					ToCurrency:   currency,
					Rate:         1.0,
					Timestamp:    time.Now(),
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
		IsHealthyFunc: func(ctx context.Context) error {
			return nil
		},
	}
}

// GetRate calls the mock implementation of GetRate.
func (m *mockExchangeRate) FetchRate(
	ctx context.Context,
	from, to string,
) (*exchange.RateInfo, error) {
	if m.GetRateFunc != nil {
		return m.GetRateFunc(ctx, from, to)
	}
	return &exchange.RateInfo{
		FromCurrency: from,
		ToCurrency:   to,
		Rate:         1.0,
		Timestamp:    time.Now(),
	}, nil
}

// GetRates calls the mock implementation of GetRates.
func (m *mockExchangeRate) FetchRates(
	ctx context.Context,
	from string,
	to []string,
) (map[string]*exchange.RateInfo, error) {
	if m.GetRatesFunc != nil {
		return m.GetRatesFunc(ctx, from, to)
	}
	result := make(map[string]*exchange.RateInfo)
	allSupportedCurrencies := []string{
		"EUR", "GBP", "JPY", "USD", "AUD",
		"CAD", "CHF", "CNY", "HKD", "NZD",
		"SGD", "ZAR",
	}

	// If 'to' is empty, return all supported currencies
	if len(to) == 0 {
		to = allSupportedCurrencies
	}

	for _, currency := range to {
		// Check if the currency is in our list of all supported currencies
		found := false
		for _, supported := range allSupportedCurrencies {
			if currency == supported {
				found = true
				break
			}
		}
		if !found {
			continue // Skip if not a supported currency
		}

		result[currency] = &exchange.RateInfo{
			FromCurrency: from,
			ToCurrency:   currency,
			Rate:         1.0,
			Timestamp:    time.Now(),
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

// Metadata calls the mock implementation of Metadata.
func (m *mockExchangeRate) Metadata() exchange.ProviderMetadata {
	if m.NameFunc != nil { // Still using NameFunc for simplicity in mock
		return exchange.ProviderMetadata{Name: m.NameFunc()}
	}
	return exchange.ProviderMetadata{Name: "mock"}
}

// CheckHealth calls the mock implementation of CheckHealth.
func (m *mockExchangeRate) CheckHealth(ctx context.Context) error {
	if m.IsHealthyFunc != nil {
		return m.IsHealthyFunc(ctx)
	}
	return nil
}

// SupportedPairs returns all supported currency pairs
func (m *mockExchangeRate) SupportedPairs() []string {
	// For simplicity, return a hardcoded list or derive from a configuration
	return []string{"USD/EUR", "EUR/USD", "USD/GBP", "GBP/USD", "USD/JPY", "JPY/USD"}
}
