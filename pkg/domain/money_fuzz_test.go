package domain_test

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
)

// FuzzNewMoney tests NewMoney invariants with random input.
func FuzzNewMoney(f *testing.F) {
	f.Add(100.0, "USD") // Seed input
	f.Add(-50.0, "EUR")
	f.Add(0.0, "JPY")
	f.Add(1e12, "ZZZ")
	f.Fuzz(func(t *testing.T, amount float64, currency currency.Code) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("NewMoney panicked: %v (amount=%v, currency=%q)", r, amount, currency)
			}
		}()
		money, err := domain.NewMoney(amount, currency)
		if err == nil {
			if !domain.IsValidCurrencyFormat(string(money.Currency())) {
				t.Errorf("Money currency is invalid: %q", money.Currency())
			}
		}
	})
}
