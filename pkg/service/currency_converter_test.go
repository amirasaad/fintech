package service_test

import (
	"errors"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures"
	"github.com/amirasaad/fintech/pkg/contracts"
)

func TestCurrencyConversion(t *testing.T) {
	tests := []struct {
		name    string
		amount  float64
		from    string
		to      string
		want    *contracts.ConversionInfo
		wantErr bool
	}{
		{"USD to EUR", 100, "USD", "EUR", &contracts.ConversionInfo{
			OriginalAmount:    100,
			OriginalCurrency:  "USD",
			ConvertedAmount:   90,
			ConvertedCurrency: "EUR",
			ConversionRate:    0.9,
		}, false},
		{"EUR to GBP", 100, "EUR", "GBP", &contracts.ConversionInfo{
			OriginalAmount:    100,
			OriginalCurrency:  "EUR",
			ConvertedAmount:   80,
			ConvertedCurrency: "GBP",
			ConversionRate:    0.8,
		}, false},
		{"Unsupported currency", 100, "USD", "XXX", nil, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			converter := fixtures.NewMockCurrencyConverter(t)
			var err error
			if test.wantErr {
				err = errors.New("test error")
			}

			converter.EXPECT().Convert(test.amount, test.from, test.to).Return(test.want, err)
			converted, err := converter.Convert(test.amount, test.from, test.to)
			if (err != nil) != test.wantErr {
				t.Errorf("Convert() error = %v, wantErr %v", err, test.wantErr)
			}
			if converted != nil {
				if converted.ConvertedAmount != test.want.ConvertedAmount {
					t.Errorf("Convert() = %v, want %v", converted, test.want)
				}
			}
		})
	}
}
