package exchange

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			name:   "successful conversion",
			amount: amount,
			to:     "EUR",
			setupMocks: func(ep *mocks.ExchangeRateProvider, rp *mocks.RegistryProvider) {
				rp.On("Get", ctx, "USD:EUR").Return(&ExchangeRateInfo{
					BaseEntity: registry.BaseEntity{},
					From:       "USD",
					To:         "EUR",
					Rate:       0.85,
				}, nil).Once()
			},
			expected: 85.0,
		},
		{
			name:   "error getting rate",
			amount: amount,
			to:     "JPY",
			setupMocks: func(ep *mocks.ExchangeRateProvider, rp *mocks.RegistryProvider) {
				ep.On("GetRate", ctx, "USD", "JPY").Return(nil, fmt.Errorf("rate not found")).Once()
				rp.On("Get", ctx, "USD:JPY").Return(nil, fmt.Errorf("rate not found")).Once()
			},
			expectedErr: "failed to get exchange rate",
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
				logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
			}

			result, rate, err := svc.Convert(ctx, tt.amount, money.Code(tt.to))

			if tt.expectedErr != "" {
				require.ErrorContains(t, err, tt.expectedErr)
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, rate)
			assert.InDelta(t, tt.expected, result.AmountFloat(), 0.0001)
			assert.InDelta(t, tt.expected/100, rate.ConversionRate, 0.0001)
		})
	}
}
