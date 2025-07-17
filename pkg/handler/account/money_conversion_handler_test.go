package account

import (
	"context"
	"errors"
	"testing"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/stretchr/testify/assert"
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

func TestMoneyConversionHandler_BusinessLogic(t *testing.T) {
	usd := currency.USD
	eur := currency.EUR
	moneyUSD, _ := money.New(100, usd)
	moneyEUR, _ := money.New(90, eur)
	convInfo := &common.ConversionInfo{OriginalAmount: 100, OriginalCurrency: string(usd), ConvertedAmount: 90, ConvertedCurrency: string(eur), ConversionRate: 0.9}

	tests := []struct {
		name        string
		input       events.MoneyCreatedEvent
		converter   *mockCurrencyConverter
		expectPub   bool
		expectMoney money.Money
		expectConv  *common.ConversionInfo
	}{
		{
			name: "no conversion needed (currencies match)",
			input: events.MoneyCreatedEvent{
				Amount:         10000,
				Currency:       string(usd),
				TargetCurrency: string(usd),
			},
			converter: &mockCurrencyConverter{
				convertFn: func(amount float64, from, to string) (*common.ConversionInfo, error) {
					return nil, nil // Should not be called
				},
			},
			expectPub:   true,
			expectMoney: moneyUSD,
			expectConv:  nil,
		},
		{
			name: "conversion needed (currencies differ)",
			input: events.MoneyCreatedEvent{
				Amount:         10000,
				Currency:       string(usd),
				TargetCurrency: string(eur),
			},
			converter: &mockCurrencyConverter{
				convertFn: func(amount float64, from, to string) (*common.ConversionInfo, error) {
					return convInfo, nil
				},
			},
			expectPub:   true,
			expectMoney: moneyEUR,
			expectConv:  convInfo,
		},
		{
			name: "converter returns error",
			input: events.MoneyCreatedEvent{
				Amount:         10000,
				Currency:       string(usd),
				TargetCurrency: string(eur),
			},
			converter: &mockCurrencyConverter{
				convertFn: func(amount float64, from, to string) (*common.ConversionInfo, error) {
					return nil, errors.New("conversion failed")
				},
			},
			expectPub:  false,
			expectConv: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := &mockEventBus{}
			handler := MoneyConversionHandler(bus, tc.converter)
			ctx := context.Background()
			handler(ctx, tc.input)
			if tc.expectPub {
				assert.Len(t, bus.published, 1)
				evt, ok := bus.published[0].(events.MoneyConvertedEvent)
				assert.True(t, ok, "should publish MoneyConvertedEvent")
				if tc.expectConv == nil {
					assert.Equal(t, tc.expectMoney.Amount(), evt.Amount)
					assert.Equal(t, tc.expectMoney.Currency().String(), evt.Currency)
					assert.Nil(t, evt.ConversionInfo)
				} else {
					assert.InEpsilon(t, tc.expectConv.ConvertedAmount, float64(evt.Amount)/100, 0.001)
					assert.Equal(t, tc.expectConv, evt.ConversionInfo)
				}
			} else {
				assert.Empty(t, bus.published)
			}
		})
	}
}
