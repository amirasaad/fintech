package conversion

import (
	"context"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockCurrencyConverter struct {
	convertFn func(amount float64, from, to string) (*common.ConversionInfo, error)
}

func (m *mockCurrencyConverter) Convert(amount float64, from, to string) (*common.ConversionInfo, error) {
	return m.convertFn(amount, from, to)
}

func (m *mockCurrencyConverter) GetRate(from, to string) (float64, error) {
	return 1.0, nil // Not used in these tests
}

func (m *mockCurrencyConverter) IsSupported(from, to string) bool {
	return true // Not used in these tests
}

func TestHandler_BusinessLogic(t *testing.T) {
	usd := currency.USD
	eur := currency.EUR
	tests := []struct {
		name        string
		input       events.CurrencyConversionRequested
		expectPub   bool
		expectMoney money.Money
		expectConv  *common.ConversionInfo
		setupMocks  func(bus *mocks.MockEventBus)
	}{
		{
			name: "conversion done",
			input: events.CurrencyConversionRequested{
				EventID:        uuid.New(),
				AccountID:      uuid.New(),
				UserID:         uuid.New(),
				Amount:         money.NewFromData(10000, string(usd)),
				SourceCurrency: string(usd),
				TargetCurrency: string(eur),
				Timestamp:      0,
			},
			expectPub:   true,
			expectMoney: money.Money{}, // Not checked in this stub
			expectConv:  nil,
			setupMocks: func(bus *mocks.MockEventBus) {
				bus.On("Publish", mock.Anything, mock.MatchedBy(func(e any) bool {
					_, ok := e.(events.CurrencyConversionDone)
					return ok
				})).Return(nil)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := mocks.NewMockEventBus(t)
			if tc.setupMocks != nil {
				tc.setupMocks(bus)
			}
			handler := Handler(bus, &mockCurrencyConverter{convertFn: func(amount float64, from, to string) (*common.ConversionInfo, error) {
				return &common.ConversionInfo{}, nil
			}}, slog.Default())
			ctx := context.Background()
			handler(ctx, tc.input)
			if tc.expectPub {
				assert.True(t, bus.AssertCalled(t, "Publish", ctx, mock.AnythingOfType("events.CurrencyConversionDone")), "should publish CurrencyConversionDone")
			} else {
				bus.AssertNotCalled(t, "Publish", ctx, mock.AnythingOfType("events.CurrencyConversionDone"))
			}
		})
	}
}
