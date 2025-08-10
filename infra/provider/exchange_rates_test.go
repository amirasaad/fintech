package provider

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/amirasaad/fintech/pkg/config"

	infra_cache "github.com/amirasaad/fintech/infra/cache"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockExchangeRateProvider is a mock implementation for testing
type MockExchangeRateProvider struct {
	mock.Mock
}

func (m *MockExchangeRateProvider) GetRate(from, to string) (*domain.ExchangeRate, error) {
	args := m.Called(from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ExchangeRate), args.Error(1)
}

func (m *MockExchangeRateProvider) GetRates(
	from string,
	to []string,
) (map[string]*domain.ExchangeRate, error) {
	args := m.Called(from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]*domain.ExchangeRate), args.Error(1)
}

func (m *MockExchangeRateProvider) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockExchangeRateProvider) IsHealthy() bool {
	args := m.Called()
	return args.Bool(0)
}

func TestExchangeRateService_GetRate_SameCurrency(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.ExchangeRate{CacheTTL: time.Minute}
	service := NewExchangeRateService(
		[]provider.ExchangeRate{},
		infra_cache.NewMemoryCache(),
		logger,
		cfg,
	)

	rate, err := service.GetRate("USD", "USD")
	require.NoError(t, err)
	assert.InEpsilon(t, 1.0, rate.Rate, 0.0001)
	assert.Equal(t, "USD", rate.FromCurrency)
	assert.Equal(t, "USD", rate.ToCurrency)
	assert.Equal(t, "internal", rate.Source)
}

func TestExchangeRateService_GetRate_FromCache(t *testing.T) {
	cache := infra_cache.NewMemoryCache()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.ExchangeRate{CacheTTL: time.Minute}
	mockProvider := &MockExchangeRateProvider{}
	mockProvider.On("Name").Return("test-provider")
	mockProvider.On("IsHealthy").Return(true)
	mockProvider.On("GetRate", "USD", "EUR").Return(&domain.ExchangeRate{
		FromCurrency: "USD",
		ToCurrency:   "EUR",
		Rate:         0.85,
	}, nil)
	service := NewExchangeRateService(
		[]provider.ExchangeRate{mockProvider},
		cache,
		logger,
		cfg,
	)

	// Create a test rate
	testRate := &domain.ExchangeRate{
		FromCurrency: "USD",
		ToCurrency:   "EUR",
		Rate:         0.85,
		LastUpdated:  time.Now(),
		Source:       "test",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	// Store in cache
	err := cache.Set("USD:EUR", testRate, 1*time.Hour)
	require.NoError(t, err)

	// Retrieve from cache
	rate, err := service.GetRate("USD", "EUR")
	require.NoError(t, err)
	assert.InEpsilon(t, 0.85, rate.Rate, 0.0001)
	assert.Equal(t, "test", rate.Source)
}

func TestExchangeRateService_GetRate_FromProvider(t *testing.T) {
	mockProvider := new(MockExchangeRateProvider)
	cache := infra_cache.NewMemoryCache()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.ExchangeRate{CacheTTL: time.Minute}
	service := NewExchangeRateService(
		[]provider.ExchangeRate{mockProvider},
		cache,
		logger,
		cfg,
	)

	// Setup mock
	expectedRate := &domain.ExchangeRate{
		FromCurrency: "USD",
		ToCurrency:   "EUR",
		Rate:         0.85,
		LastUpdated:  time.Now(),
		Source:       "mock-provider",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	mockProvider.On("Name").Return("mock-provider")
	mockProvider.On("IsHealthy").Return(true)
	mockProvider.On("GetRate", "USD", "EUR").Return(expectedRate, nil)

	// Get rate
	rate, err := service.GetRate("USD", "EUR")
	require.NoError(t, err)
	assert.InEpsilon(t, 0.85, rate.Rate, 0.0001)
	assert.Equal(t, "mock-provider", rate.Source)

	// Verify it was cached
	cachedRate, err := cache.Get("USD:EUR")
	require.NoError(t, err)
	assert.InEpsilon(t, 0.85, cachedRate.Rate, 0.0001)

	mockProvider.AssertExpectations(t)
}

func TestExchangeRateService_GetRate_ProviderUnhealthy(t *testing.T) {
	mockProvider := new(MockExchangeRateProvider)
	cache := infra_cache.NewMemoryCache()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.ExchangeRate{CacheTTL: time.Minute}
	service := NewExchangeRateService(
		[]provider.ExchangeRate{mockProvider},
		cache,
		logger,
		cfg,
	)

	// Setup mock to be unhealthy
	mockProvider.On("Name").Return("mock-provider")
	mockProvider.On("IsHealthy").Return(false)

	// Try to get rate
	rate, err := service.GetRate("USD", "EUR")
	require.Error(t, err)
	assert.Nil(t, rate)
	assert.Equal(t, domain.ErrExchangeRateUnavailable, err)

	mockProvider.AssertExpectations(t)
}

func TestExchangeRateService_GetRate_ProviderError(t *testing.T) {
	mockProvider := new(MockExchangeRateProvider)
	cache := infra_cache.NewMemoryCache()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.ExchangeRate{CacheTTL: time.Minute}
	service := NewExchangeRateService(
		[]provider.ExchangeRate{mockProvider},
		cache,
		logger,
		cfg,
	)

	// Setup mock to return error
	mockProvider.On("Name").Return("mock-provider")
	mockProvider.On("IsHealthy").Return(true)
	mockProvider.On("GetRate", "USD", "EUR").Return(nil, assert.AnError)

	// Try to get rate
	rate, err := service.GetRate("USD", "EUR")
	require.Error(t, err)
	assert.Nil(t, rate)
	assert.Equal(t, domain.ErrExchangeRateUnavailable, err)

	mockProvider.AssertExpectations(t)
}

func TestExchangeRateService_GetRate_InvalidRate(t *testing.T) {
	mockProvider := new(MockExchangeRateProvider)
	cache := infra_cache.NewMemoryCache()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.ExchangeRate{CacheTTL: time.Minute}
	service := NewExchangeRateService(
		[]provider.ExchangeRate{mockProvider},
		cache,
		logger,
		cfg,
	)

	// Setup mock to return invalid rate
	invalidRate := &domain.ExchangeRate{
		FromCurrency: "USD",
		ToCurrency:   "EUR",
		Rate:         -1.0, // Invalid negative rate
		LastUpdated:  time.Now(),
		Source:       "mock-provider",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	mockProvider.On("Name").Return("mock-provider")
	mockProvider.On("IsHealthy").Return(true)
	mockProvider.On("GetRate", "USD", "EUR").Return(invalidRate, nil)

	// Try to get rate
	rate, err := service.GetRate("USD", "EUR")
	require.Error(t, err)
	assert.Nil(t, rate)
	assert.Equal(t, domain.ErrExchangeRateUnavailable, err)

	mockProvider.AssertExpectations(t)
}

func TestExchangeRateService_GetRates_MultipleCurrencies(t *testing.T) {
	mockProvider := new(MockExchangeRateProvider)
	cache := infra_cache.NewMemoryCache()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.ExchangeRate{CacheTTL: time.Minute}
	service := NewExchangeRateService(
		[]provider.ExchangeRate{mockProvider},
		cache,
		logger,
		cfg,
	)

	// Setup mock
	expectedRates := map[string]*domain.ExchangeRate{
		"EUR": {
			FromCurrency: "USD",
			ToCurrency:   "EUR",
			Rate:         0.85,
			LastUpdated:  time.Now(),
			Source:       "mock-provider",
			ExpiresAt:    time.Now().Add(1 * time.Hour),
		},
		"GBP": {
			FromCurrency: "USD",
			ToCurrency:   "GBP",
			Rate:         0.75,
			LastUpdated:  time.Now(),
			Source:       "mock-provider",
			ExpiresAt:    time.Now().Add(1 * time.Hour),
		},
	}

	mockProvider.On("IsHealthy").Return(true)
	mockProvider.On("GetRates", "USD", []string{"EUR", "GBP"}).Return(expectedRates, nil)

	// Get rates
	rates, err := service.GetRates("USD", []string{"EUR", "GBP"})
	require.NoError(t, err)
	assert.Len(t, rates, 2)
	assert.InEpsilon(t, 0.85, rates["EUR"].Rate, 0.0001)
	assert.InEpsilon(t, 0.75, rates["GBP"].Rate, 0.0001)

	mockProvider.AssertExpectations(t)

}
