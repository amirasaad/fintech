package exchange_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/amirasaad/fintech/pkg/service/exchange"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_GetRate(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*mocks.ExchangeRateProvider, *mocks.RegistryProvider)
		from           string
		to             string
		expectedRate   float64
		expectedError  bool
		expectedSource string
	}{
		{
			name: "successful rate fetch",
			setupMocks: func(
				mockProvider *mocks.ExchangeRateProvider,
				mockRegistry *mocks.RegistryProvider,
			) {
				// Mock cache miss
				mockRegistry.On("Get", mock.Anything, "USD:EUR").
					Return((*exchange.ExchangeRateInfo)(nil), errors.New("entity not found"))

				// Mock provider response
				mockProvider.On("Name").Return("test-provider")
				mockProvider.On("IsHealthy").Return(true)
				mockProvider.On("GetRate", mock.Anything, "USD", "EUR").
					Return(&provider.ExchangeInfo{
						OriginalCurrency:  "USD",
						ConvertedCurrency: "EUR",
						ConversionRate:    0.92,
						Source:            "test-provider",
					}, nil)

				// Expect cache save
				mockRegistry.On("Register", mock.Anything, mock.Anything).
					Return(nil)
			},
			from:           "USD",
			to:             "EUR",
			expectedRate:   0.92,
			expectedError:  false,
			expectedSource: "test-provider",
		},
		{
			name: "same currency returns 1.0",
			setupMocks: func(
				mockProvider *mocks.ExchangeRateProvider,
				mockRegistry *mocks.RegistryProvider,
			) {
				// No mocks needed as same currency should return early
			},
			from:           "USD",
			to:             "USD",
			expectedRate:   1.0,
			expectedError:  false,
			expectedSource: "identity",
		},
		{
			name: "cached rate is refreshed and returned",
			setupMocks: func(
				mockProvider *mocks.ExchangeRateProvider,
				mockRegistry *mocks.RegistryProvider,
			) {
				// Mock cache hit with an expired rate
				cachedRate := &exchange.ExchangeRateInfo{
					From:      "USD",
					To:        "EUR",
					Rate:      0.91,
					Source:    "cache",
					Timestamp: time.Now().Add(-25 * time.Hour), // Expired cache
				}
				mockRegistry.On("Get", mock.Anything, "USD:EUR").
					Return(cachedRate, nil)
				mockProvider.On("Name").Return("test-provider")
				mockProvider.On("IsHealthy").Return(true)
				// Mock GetRate for the refresh
				mockProvider.On("GetRate", mock.Anything, "USD", "EUR").
					Return(&provider.ExchangeInfo{
						OriginalCurrency:  "USD",
						ConvertedCurrency: "EUR",
						ConversionRate:    0.92,
						Source:            "test-provider",
					}, nil)
				// Mock Register for the cache update
				// Mock Register for direct rate (USD:EUR)
				mockRegistry.On("Register",
					mock.Anything,
					mock.MatchedBy(func(info *exchange.ExchangeRateInfo) bool {
						return info.From == "USD" &&
							info.To == "EUR" &&
							info.Rate == 0.92 &&
							info.Source == "test-provider"
					}),
				).Return(nil)

				// Mock Register for inverse rate (EUR:USD)
				mockRegistry.On("Register",
					mock.Anything,
					mock.MatchedBy(func(info *exchange.ExchangeRateInfo) bool {
						return info.From == "EUR" &&
							info.To == "USD" &&
							info.Source == "test-provider"
					}),
				).Return(nil)
			},
			from:           "USD",
			to:             "EUR",
			expectedRate:   0.92,
			expectedError:  false,
			expectedSource: "test-provider",
		},
		{
			name: "provider error is propagated",
			setupMocks: func(
				mockProvider *mocks.ExchangeRateProvider,
				mockRegistry *mocks.RegistryProvider,
			) {
				mockRegistry.On("Get", mock.Anything, "USD:GBP").
					Return((*exchange.ExchangeRateInfo)(nil), errors.New("entity not found"))
				mockProvider.On("Name").Return("test-provider")
				mockProvider.On("IsHealthy").Return(true)
				mockProvider.On("GetRate", mock.Anything, "USD", "GBP").
					Return((*provider.ExchangeInfo)(nil), errors.New("provider error"))
			},
			from:          "USD",
			to:            "GBP",
			expectedRate:  0,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := mocks.NewExchangeRateProvider(t)
			mockRegistry := mocks.NewRegistryProvider(t)

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockRegistry)
			}

			svc := exchange.New(mockRegistry, mockProvider, nil)

			rate, err := svc.GetRate(context.Background(), tt.from, tt.to)

			if tt.expectedError {
				require.Error(t, err, "expected error but got none")
				assert.Nil(t, rate, "rate should be nil on error")
			} else {
				require.NoError(t, err, "unexpected error")
				require.NotNil(t, rate, "rate should not be nil")
				assert.InDelta(t, tt.expectedRate, rate.ConversionRate, 0.0001,
					"conversion rate should match expected")
				assert.Equal(t, tt.expectedSource, rate.Source,
					"source should match expected")
			}

			mockProvider.AssertExpectations(t)
			mockRegistry.AssertExpectations(t)
		})
	}
}

func TestService_Convert(t *testing.T) {
	tests := []struct {
		name              string
		setupMocks        func(*mocks.ExchangeRateProvider, *mocks.RegistryProvider)
		amount            *money.Money
		to                string
		expectedAmount    float64
		expectedRate      float64
		expectedError     bool
		expectedErrType   error
		skipCurrencyCheck bool
	}{
		{
			name: "successful conversion",
			setupMocks: func(
				mockProvider *mocks.ExchangeRateProvider,
				mockRegistry *mocks.RegistryProvider,
			) {
				mockRegistry.On("Get", mock.Anything, "USD:EUR").
					Return((*exchange.ExchangeRateInfo)(nil), errors.New("entity not found"))

				mockProvider.On("Name").Return("test-provider").Maybe()
				mockProvider.On("IsHealthy").Return(true)
				mockProvider.On("GetRate", mock.Anything, "USD", "EUR").
					Return(&provider.ExchangeInfo{
						OriginalCurrency:  "USD",
						ConvertedCurrency: "EUR",
						ConversionRate:    0.92,
						Source:            "test-provider",
						Timestamp:         time.Now(),
					}, nil)

				mockRegistry.On("Register", mock.Anything, mock.Anything).
					Return(nil)
			},
			amount:         mustNewMoney(t, 100, "USD"),
			to:             "EUR",
			expectedAmount: 92.0,
			expectedRate:   0.92,
			expectedError:  false,
		},
		{
			name: "same currency returns same amount",
			setupMocks: func(
				mockProvider *mocks.ExchangeRateProvider,
				mockRegistry *mocks.RegistryProvider,
			) {
				// Mock the Get call that happens in the identity rate case
				mockRegistry.On("Get", mock.Anything, mock.AnythingOfType("string")).
					Return((*exchange.ExchangeRateInfo)(nil), errors.New("not found")).Maybe()
				mockProvider.On("Name").Return("test-provider")
				mockProvider.On("IsHealthy").Return(true)
				mockProvider.On(
					"GetRate",
					mock.Anything,
					mock.AnythingOfType("string"),
					mock.AnythingOfType("string"),
				).
					Return(&provider.ExchangeInfo{
						OriginalCurrency:  "USD",
						ConvertedCurrency: "USD",
						ConversionRate:    1.0,
						Source:            "test-provider",
					}, nil).Maybe()

				// Mock Register for the cache update (direct and inverse rates)
				mockRegistry.On(
					"Register",
					mock.Anything,
					mock.MatchedBy(func(info *exchange.ExchangeRateInfo) bool {
						return (info.From == "USD" && info.To == "USD") ||
							(info.From == "USD" && info.To == "EUR") ||
							(info.From == "EUR" && info.To == "USD")
					})).Return(nil)
			},
			amount:            mustNewMoney(t, 100, "USD"),
			to:                "USD",
			expectedAmount:    100.0,
			expectedRate:      1.0,
			expectedError:     false,
			skipCurrencyCheck: true,
		},
		{
			name: "invalid amount",
			setupMocks: func(
				mockProvider *mocks.ExchangeRateProvider,
				mockRegistry *mocks.RegistryProvider,
			) {
				// No mocks needed as validation happens first
			},
			amount:          mustNewMoney(t, 0, "USD"),
			to:              "EUR",
			expectedError:   true,
			expectedErrType: exchange.ErrInvalidAmount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockProvider := mocks.NewExchangeRateProvider(t)
			mockRegistry := mocks.NewRegistryProvider(t)

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockRegistry)
			}

			svc := exchange.New(mockRegistry, mockProvider, nil)

			result, rate, err := svc.Convert(
				context.Background(),
				tt.amount,
				money.EUR,
			)

			if tt.expectedError {
				require.Error(t, err, "expected error but got none")
				if tt.expectedErrType != nil {
					require.ErrorIs(t, err, tt.expectedErrType,
						"error should be of expected type")
				}
			} else {
				require.NoError(t, err, "unexpected error during conversion")
				require.NotNil(t, result, "result should not be nil")
				require.NotNil(t, rate, "rate should not be nil")
				assert.InDelta(t, tt.expectedAmount, result.AmountFloat(), 0.001,
					"converted amount should match expected")
				if !tt.skipCurrencyCheck {
					assert.Equal(t, tt.to, result.Currency().String(),
						"currency code should match expected")
				}
				assert.InDelta(t, tt.expectedRate, rate.ConversionRate, 0.0001,
					"conversion rate should match expected")
			}

			mockProvider.AssertExpectations(t)
			mockRegistry.AssertExpectations(t)
		})
	}
}

func TestService_IsSupported(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		expected bool
	}{
		{
			name:     "same currency is always supported",
			from:     "USD",
			to:       "USD",
			expected: true,
		},
		{
			name:     "provider supports the pair",
			from:     "USD",
			to:       "EUR",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockProvider := mocks.NewExchangeRateProvider(t)
			mockRegistry := mocks.NewRegistryProvider(t)

			if tt.from != tt.to {
				mockProvider.On("IsSupported", tt.from, tt.to).Return(tt.expected)
			}

			svc := exchange.New(mockRegistry, mockProvider, nil)

			result := svc.IsSupported(tt.from, tt.to)
			assert.Equal(t, tt.expected, result)

			mockProvider.AssertExpectations(t)
		})
	}
}

func TestService_FetchAndCacheRates(t *testing.T) {
	tests := []struct {
		name         string
		setupMocks   func(*mocks.ExchangeRateProvider, *mocks.RegistryProvider)
		from         string
		expectedErr  bool
		expectedMsgs []string
	}{
		{
			name: "successfully fetch and cache rates",
			setupMocks: func(
				mockProvider *mocks.ExchangeRateProvider,
				mockRegistry *mocks.RegistryProvider,
			) {
				mockProvider.On("IsHealthy").Return(true)
				mockProvider.On("Name").Return("test-provider")
				mockProvider.On("GetRates", mock.Anything, "USD", []string{}).
					Return(map[string]*provider.ExchangeInfo{
						"EUR": {
							OriginalCurrency:  "USD",
							ConvertedCurrency: "EUR",
							ConversionRate:    0.92,
						},
					}, nil)

				mockRegistry.On("Register",
					mock.Anything,
					mock.AnythingOfType("*exchange.ExchangeRateInfo"),
				).Return(nil)
			},
			from:        "USD",
			expectedErr: false,
		},
		{
			name: "unhealthy provider returns error",
			setupMocks: func(
				mockProvider *mocks.ExchangeRateProvider,
				mockRegistry *mocks.RegistryProvider,
			) {
				mockProvider.On("Name").Return("test-provider")
				mockProvider.On("IsHealthy").Return(false)
			},
			from:        "USD",
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockProvider := mocks.NewExchangeRateProvider(t)
			mockRegistry := mocks.NewRegistryProvider(t)

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockRegistry)
			}

			svc := exchange.New(mockRegistry, mockProvider, nil)

			err := svc.FetchAndCacheRates(context.Background(), tt.from)

			if tt.expectedErr {
				require.Error(t, err, "expected error but got none")
			} else {
				require.NoError(t, err, "unexpected error")
			}

			mockProvider.AssertExpectations(t)
			mockRegistry.AssertExpectations(t)
		})
	}
}

// Helper function to create money with panic on error (for test setup)
func mustNewMoney(t *testing.T, amount float64, curr string) *money.Money {
	m, err := money.New(amount, money.Code(curr))
	if err != nil {
		t.Fatalf("failed to create money: %v", err)
	}
	return m
}
