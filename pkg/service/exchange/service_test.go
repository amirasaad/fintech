package exchange

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_IsSupported(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		setup    func(*mocks.ExchangeRateProvider)
		expected bool
	}{
		{
			name:     "same currency",
			from:     "USD",
			to:       "USD",
			setup:    func(m *mocks.ExchangeRateProvider) {},
			expected: true,
		},
		{
			name: "supported currency pair",
			from: "USD",
			to:   "EUR",
			setup: func(m *mocks.ExchangeRateProvider) {
				m.On("IsSupported", "USD", "EUR").Return(true).Once()
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := mocks.NewExchangeRateProvider(t)
			mockRegistry := mocks.NewRegistryProvider(t)
			tt.setup(mockProvider)

			svc := New(mockRegistry, mockProvider, nil)
			result := svc.IsSupported(tt.from, tt.to)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestService_GetRate(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		from        string
		to          string
		setupMocks  func(*mocks.ExchangeRateProvider, *mocks.RegistryProvider)
		expected    *provider.ExchangeInfo
		expectedErr string
	}{
		{
			name:       "same currency",
			from:       "USD",
			to:         "USD",
			setupMocks: func(ep *mocks.ExchangeRateProvider, rp *mocks.RegistryProvider) {},
			expected: &provider.ExchangeInfo{
				OriginalCurrency:  "USD",
				ConvertedCurrency: "USD",
				ConversionRate:    1.0,
				Source:            "identity",
			},
		},
		{
			name: "from cache",
			from: "USD",
			to:   "EUR",
			setupMocks: func(ep *mocks.ExchangeRateProvider, rp *mocks.RegistryProvider) {
				rp.On("Get", ctx, "USD:EUR").Return(&ExchangeRateInfo{
					From:   "USD",
					To:     "EUR",
					Rate:   0.85,
					Source: "cache",
				}, nil).Once()
			},
			expected: &provider.ExchangeInfo{
				OriginalCurrency:  "USD",
				ConvertedCurrency: "EUR",
				ConversionRate:    0.85,
				Source:            "cache",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := mocks.NewExchangeRateProvider(t)
			mockRegistry := mocks.NewRegistryProvider(t)
			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockRegistry)
			}

			svc := New(mockRegistry, mockProvider, nil)
			result, err := svc.GetRate(ctx, tt.from, tt.to)

			if tt.expectedErr != "" {
				assert.ErrorContains(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected.OriginalCurrency, result.OriginalCurrency)
			assert.Equal(t, tt.expected.ConvertedCurrency, result.ConvertedCurrency)
			assert.InDelta(t, tt.expected.ConversionRate, result.ConversionRate, 0.0001)
			assert.Equal(t, tt.expected.Source, result.Source)
		})
	}
}

func TestService_Convert(t *testing.T) {
	ctx := context.Background()
	amount, _ := money.New(100, "USD")

	tests := []struct {
		name        string
		amount      *money.Money
		to          string
		setupMocks  func(*mocks.ExchangeRateProvider, *mocks.RegistryProvider)
		expected    float64
		expectedErr string
	}{
		{
			name:       "same currency",
			amount:     amount,
			to:         "USD",
			setupMocks: func(ep *mocks.ExchangeRateProvider, rp *mocks.RegistryProvider) {},
			expected:   100.0,
		},
		{
			name:   "convert USD to EUR",
			amount: amount,
			to:     "EUR",
			setupMocks: func(ep *mocks.ExchangeRateProvider, rp *mocks.RegistryProvider) {
				rp.On("Get", ctx, "USD:EUR").Return(&ExchangeRateInfo{
					From: "USD",
					To:   "EUR",
					Rate: 0.85,
				}, nil).Once()
			},
			expected: 85.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := mocks.NewExchangeRateProvider(t)
			mockRegistry := mocks.NewRegistryProvider(t)
			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockRegistry)
			}

			svc := New(mockRegistry, mockProvider, nil)
			result, _, err := svc.Convert(ctx, tt.amount, money.Code(tt.to))

			if tt.expectedErr != "" {
				assert.ErrorContains(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.InDelta(t, tt.expected, result.AmountFloat(), 0.0001)
		})
	}
}

func TestService_processAndCacheRate(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name        string
		from        string
		to          string
		rate        *provider.ExchangeInfo
		setupMocks  func(*mocks.ExchangeRateProvider, *mocks.RegistryProvider)
		expectError bool
	}{
		{
			name: "valid rate",
			from: "USD",
			to:   "EUR",
			rate: &provider.ExchangeInfo{
				OriginalCurrency:  "USD",
				ConvertedCurrency: "EUR",
				ConversionRate:    0.85,
				Source:            "test",
			},
			setupMocks: func(mp *mocks.ExchangeRateProvider, mr *mocks.RegistryProvider) {
				// Called twice: once for direct rate, once for inverse
				mp.On("Name").Return("test").Twice()
				mr.On("Register", ctx, mock.MatchedBy(func(entity any) bool {
					rateInfo, ok := entity.(*ExchangeRateInfo)
					if !ok {
						return false
					}
					// Match either direct or inverse rate
					return (rateInfo.From == "USD" &&
						rateInfo.To == "EUR" &&
						rateInfo.Rate == 0.85) ||
						(rateInfo.From == "EUR" &&
							rateInfo.To == "USD" &&
							rateInfo.Rate > 1.0)
				})).Return(nil).Twice() // Expect two calls: direct and inverse rates
			},
			expectError: false,
		},
		{
			name: "nil rate",
			from: "USD",
			to:   "EUR",
			rate: nil,
			setupMocks: func(mp *mocks.ExchangeRateProvider, mr *mocks.RegistryProvider) {
				// The logger will call Name() once when logging the error
				mp.On("Name").Return("test").Twice()
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := mocks.NewExchangeRateProvider(t)
			mockRegistry := mocks.NewRegistryProvider(t)

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockRegistry)
			}

			svc := &Service{
				provider: mockProvider,
				registry: mockRegistry,
				logger:   logger,
			}

			svc.processAndCacheRate(tt.from, tt.to, tt.rate)

			// Verify all expectations were met
			mockProvider.AssertExpectations(t)
			mockRegistry.AssertExpectations(t)
		})
	}
}
