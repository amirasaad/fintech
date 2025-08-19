package exchange

import (
	"context"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_IsSupported_SameCurrency(t *testing.T) {
	// Setup
	mockProvider := mocks.NewExchangeRateProvider(t)
	mockRegistry := mocks.NewRegistryProvider(t)
	svc := New(mockRegistry, mockProvider, nil)

	// Execute
	result := svc.IsSupported("USD", "USD")

	// Verify
	assert.True(t, result, "same currency should always be supported")
}

func TestService_FetchAndCacheRates_UnhealthyProvider(t *testing.T) {
	// Setup
	mockProvider := mocks.NewExchangeRateProvider(t)
	mockRegistry := mocks.NewRegistryProvider(t)
	svc := New(mockRegistry, mockProvider, nil)

	// Mock the Get call in shouldFetchNewRates - return nil to indicate no last_updated timestamp
	mockRegistry.On("Get", mock.Anything, "exr:last_updated").
		Return(nil, nil).Once()

	// Mock provider health check
	mockProvider.On("IsHealthy").Return(false).Once()
	mockProvider.On("Name").Return("test-provider").Once()

	// Execute
	err := svc.FetchAndCacheRates(context.Background(), "USD")

	// Verify
	require.Error(t, err, "should return error for unhealthy provider")
	assert.Contains(t, err.Error(), "provider test-provider is unhealthy")

}
