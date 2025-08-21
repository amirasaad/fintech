package exchange

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/amirasaad/fintech/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestValidateAmount(t *testing.T) {
	// Helper to create money amount or fail the test
	newMoney := func(amount float64, currency string) *money.Money {
		m, err := money.New(amount, currency)
		require.NoError(t, err, "failed to create money amount")
		return m
	}

	tests := []struct {
		name        string
		amount      *money.Money
		expectedErr error
	}{
		{
			name:        "nil amount",
			amount:      nil,
			expectedErr: errors.New("amount cannot be nil"),
		},
		{
			name:        "zero amount",
			amount:      newMoney(0, "USD"),
			expectedErr: ErrInvalidAmount,
		},
		{
			name:        "negative amount",
			amount:      newMoney(-100, "USD"),
			expectedErr: ErrInvalidAmount,
		},
		{
			name:        "valid positive amount",
			amount:      newMoney(100, "USD"),
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAmount(tt.amount)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_getRateFromCache(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		setupMock     func(*mocks.RegistryProvider)
		from          string
		to            string
		expectedRate  *provider.ExchangeInfo
		expectedFound bool
	}{
		{
			name: "cache miss - registry error",
			setupMock: func(m *mocks.RegistryProvider) {
				m.On("Get", ctx, "USD:EUR").
					Return(nil, errors.New("registry error"))
			},
			from:          "USD",
			to:            "EUR",
			expectedRate:  nil,
			expectedFound: false,
		},
		{
			name: "cache miss - not found",
			setupMock: func(m *mocks.RegistryProvider) {
				m.On("Get", ctx, "USD:EUR").
					Return(nil, nil)
			},
			from:          "USD",
			to:            "EUR",
			expectedRate:  nil,
			expectedFound: false,
		},
		{
			name: "cache hit - valid rate",
			setupMock: func(m *mocks.RegistryProvider) {
				rateInfo := &ExchangeRateInfo{
					From:      "USD",
					To:        "EUR",
					Rate:      0.85,
					Source:    "test",
					Timestamp: time.Now(),
				}
				m.On("Get", ctx, "USD:EUR").
					Return(rateInfo, nil)
			},
			from: "USD",
			to:   "EUR",
			expectedRate: &provider.ExchangeInfo{
				OriginalCurrency:  "USD",
				ConvertedCurrency: "EUR",
				ConversionRate:    0.85,
				Source:            "test",
			},
			expectedFound: true,
		},
		{
			name: "cache miss - wrong entity type",
			setupMock: func(m *mocks.RegistryProvider) {
				// Create a mock entity that implements registry.Entity
				// but isn't an ExchangeRateInfo
				mockEntity := registry.NewBaseEntity("test-id", "test-name")
				m.On("Get", ctx, "USD:GBP").Return(mockEntity, nil)
			},
			from:          "USD",
			to:            "GBP",
			expectedRate:  nil,
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRegistry := mocks.NewRegistryProvider(t)
			tt.setupMock(mockRegistry)

			svc := &Service{
				registry: mockRegistry,
				logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
			}

			rate, found := svc.getRateFromCache(ctx, tt.from, tt.to)

			if tt.expectedFound {
				assert.True(t, found)
				assert.NotNil(t, rate)
				assert.Equal(t, tt.expectedRate.OriginalCurrency, rate.OriginalCurrency)
				assert.Equal(t, tt.expectedRate.ConvertedCurrency, rate.ConvertedCurrency)
				assert.InDelta(t, tt.expectedRate.ConversionRate,
					rate.ConversionRate, 0.0001,
					"conversion rate should match")
				assert.Equal(t, tt.expectedRate.Source, rate.Source)
			} else {
				assert.False(t, found)
				assert.Nil(t, rate)
			}

			mockRegistry.AssertExpectations(t)
		})
	}
}

func TestService_NameAndHealth(t *testing.T) {
	// Setup test service
	svc := &Service{}

	// Test Name method
	t.Run("Name returns correct service name", func(t *testing.T) {
		name := svc.Name()
		assert.Equal(t, "ExchangeService", name, "Name() should return 'ExchangeService'")
	})

	// Test IsHealthy method
	t.Run("IsHealthy returns true", func(t *testing.T) {
		isHealthy := svc.IsHealthy()
		assert.True(t, isHealthy, "IsHealthy() should always return true")
	})
}

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

	// Helper to create a test service with proper mocks
	newTestService := func(t *testing.T, withProvider bool) (
		*Service,
		*mocks.ExchangeRateProvider,
		*mocks.RegistryProvider,
	) {
		var mockProvider *mocks.ExchangeRateProvider
		mockRegistry := mocks.NewRegistryProvider(t)

		if withProvider {
			mockProvider = mocks.NewExchangeRateProvider(t)
			mockProvider.On("Name").Return("test-provider").Maybe()

			svc := &Service{
				provider: mockProvider,
				registry: mockRegistry,
				logger:   slog.Default(),
			}
			return svc, mockProvider, mockRegistry
		}

		// Return service with nil provider for tests that don't need a provider
		svc := &Service{
			provider: nil,
			registry: mockRegistry,
			logger:   slog.Default(),
		}
		return svc, nil, mockRegistry
	}

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
			name: "from cache with full metadata",
			from: "USD",
			to:   "EUR",
			setupMocks: func(ep *mocks.ExchangeRateProvider, rp *mocks.RegistryProvider) {
				rp.On("Get", ctx, "USD:EUR").Return(&ExchangeRateInfo{
					BaseEntity: *registry.NewBaseEntity("USD:EUR", "USD to EUR"),
					From:       "USD",
					To:         "EUR",
					Rate:       0.85,
					Source:     "cache",
					Timestamp:  time.Now().Add(-30 * time.Minute),
				}, nil).Once()
			},
			expected: &provider.ExchangeInfo{
				OriginalCurrency:  "USD",
				ConvertedCurrency: "EUR",
				ConversionRate:    0.85,
				Source:            "cache",
			},
		},
		{
			name: "cache returns valid rate",
			from: "USD",
			to:   "GBP",
			setupMocks: func(ep *mocks.ExchangeRateProvider, rp *mocks.RegistryProvider) {
				// Return a valid cached rate
				rp.On("Get", ctx, "USD:GBP").Return(&ExchangeRateInfo{
					BaseEntity: *registry.NewBaseEntity("USD:GBP", "USD to GBP"),
					From:       "USD",
					To:         "GBP",
					Rate:       0.75,
					Source:     "test-provider",
					Timestamp:  time.Now(),
				}, nil).Once()
				// No need to set Register expectation for cache hit
			},
			expected: &provider.ExchangeInfo{
				OriginalCurrency:  "USD",
				ConvertedCurrency: "GBP",
				ConversionRate:    0.75,
				Source:            "test-provider",
			},
		},
		{
			name: "cache miss with provider fallback",
			from: "USD",
			to:   "CAD",
			setupMocks: func(ep *mocks.ExchangeRateProvider, rp *mocks.RegistryProvider) {
				// Cache miss
				rp.On("Get", ctx, "USD:CAD").Return(nil, registry.ErrNotFound).Once()
				// Should fall back to provider
				ep.On("GetRate", ctx, "USD", "CAD").Return(&provider.ExchangeInfo{
					OriginalCurrency:  "USD",
					ConvertedCurrency: "CAD",
					ConversionRate:    1.25,
					Source:            "test-provider",
				}, nil).Once()
				// We don't need to test the exact Register call, just that it happens
				rp.On("Register", ctx, mock.Anything).Return(nil).Maybe()
			},
			expected: &provider.ExchangeInfo{
				OriginalCurrency:  "USD",
				ConvertedCurrency: "CAD",
				ConversionRate:    1.25,
				Source:            "test-provider",
			},
		},
		{
			name:        "empty from currency",
			from:        "",
			to:          "USD",
			setupMocks:  func(ep *mocks.ExchangeRateProvider, rp *mocks.RegistryProvider) {},
			expected:    nil,
			expectedErr: "invalid currency codes: from='', to='USD'",
		},
		{
			name:        "empty to currency",
			from:        "USD",
			to:          "",
			setupMocks:  func(ep *mocks.ExchangeRateProvider, rp *mocks.RegistryProvider) {},
			expected:    nil,
			expectedErr: "invalid currency codes: from='USD', to=''",
		},
		{
			name: "cache miss with error, fallback to provider",
			from: "USD",
			to:   "GBP",
			setupMocks: func(ep *mocks.ExchangeRateProvider, rp *mocks.RegistryProvider) {
				// Cache miss with error
				rp.On("Get", ctx, "USD:GBP").Return(nil, registry.ErrNotFound).Once()
				// Provider returns rate
				ep.On("GetRate", ctx, "USD", "GBP").Return(&provider.ExchangeInfo{
					OriginalCurrency:  "USD",
					ConvertedCurrency: "GBP",
					ConversionRate:    0.75,
					Source:            "test-provider",
				}, nil).Once()
				// We don't need to test the exact Register call, just that it happens
				rp.On("Register", ctx, mock.Anything).Return(nil).Maybe()
			},
			expected: &provider.ExchangeInfo{
				OriginalCurrency:  "USD",
				ConvertedCurrency: "GBP",
				ConversionRate:    0.75,
				Source:            "test-provider",
			},
		},
		{
			name: "cache error with provider fallback",
			from: "USD",
			to:   "JPY",
			setupMocks: func(ep *mocks.ExchangeRateProvider, rp *mocks.RegistryProvider) {
				// Cache error
				rp.On("Get", ctx, "USD:JPY").Return(nil, errors.New("cache error")).Once()
				// Should fallback to provider for USD/JPY
				ep.On("GetRate", ctx, "USD", "JPY").Return(&provider.ExchangeInfo{
					OriginalCurrency:  "USD",
					ConvertedCurrency: "JPY",
					ConversionRate:    150.0,
					Source:            "test-provider",
				}, nil).Once()
				// We don't need to test the exact Register call, just that it happens
				rp.On("Register", ctx, mock.Anything).Return(nil).Maybe()
			},
			expected: &provider.ExchangeInfo{
				OriginalCurrency:  "USD",
				ConvertedCurrency: "JPY",
				ConversionRate:    150.0,
				Source:            "test-provider",
			},
		},
		{
			name: "no provider available",
			from: "USD",
			to:   "CAD",
			setupMocks: func(ep *mocks.ExchangeRateProvider, rp *mocks.RegistryProvider) {
				// Cache miss
				rp.On("Get", ctx, "USD:CAD").Return(nil, registry.ErrNotFound).Once()
				// No provider calls should be made - ep is nil in this case
				// No need to set up any expectations on ep since it's nil
			},
			expected:    nil,
			expectedErr: "no exchange rate providers available",
		},
		{
			name: "provider returns error",
			from: "USD",
			to:   "AUD",
			setupMocks: func(ep *mocks.ExchangeRateProvider, rp *mocks.RegistryProvider) {
				rp.On("Get", ctx, "USD:AUD").Return(
					nil,
					registry.ErrNotFound,
				).Once()
				ep.On("GetRate", ctx, "USD", "AUD").Return(
					nil,
					errors.New("provider error"),
				).Once()
			},
			expected:    nil,
			expectedErr: "failed to fetch rates from provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new test service with a provider for all tests
			// except "no provider available"
			hasProvider := tt.name != "no provider available"
			svc, mockProvider, mockRegistry := newTestService(t, hasProvider)

			// Clear any existing mock expectations
			mockRegistry.ExpectedCalls = nil

			if mockProvider != nil {
				mockProvider.ExpectedCalls = nil
				mockProvider.On("Name").Return("test-provider").Maybe()
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockRegistry)
			}

			result, err := svc.GetRate(ctx, tt.from, tt.to)

			if tt.expectedErr != "" {
				assert.ErrorContains(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(
				t,
				tt.expected.OriginalCurrency,
				result.OriginalCurrency,
			)
			assert.Equal(
				t,
				tt.expected.ConvertedCurrency,
				result.ConvertedCurrency,
			)
			assert.InDelta(
				t,
				tt.expected.ConversionRate,
				result.ConversionRate,
				0.0001,
			)
			assert.Equal(
				t,
				tt.expected.Source,
				result.Source,
			)
		})
	}
}
