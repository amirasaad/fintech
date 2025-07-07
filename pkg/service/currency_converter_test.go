package service_test

import (
	"errors"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures"
)

func TestCurrencyConversion(t *testing.T) {
	tests := []struct {
		name    string
		amount  float64
		from    string
		to      string
		rate    float64
		want    float64
		wantErr bool
	}{
		{"USD to EUR", 100, "USD", "EUR", 0.9, 90, false},
		{"EUR to GBP", 100, "EUR", "GBP", 0.8, 80, false},
		{"Unsupported currency", 100, "USD", "XXX", 0, 0, true},
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
			if converted != test.want {
				t.Errorf("Convert() = %v, want %v", converted, test.want)
			}
		})
	}
}
