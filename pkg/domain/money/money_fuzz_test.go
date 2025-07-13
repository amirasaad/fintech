package money_test

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/money"
)

// FuzzNewMoney tests NewMoney invariants with random input.
func FuzzNewMoney(f *testing.F) {
	f.Add(100.0, "USD") // Seed input
	f.Add(-50.0, "EUR")
	f.Add(0.0, "JPY")
	f.Add(1e12, "ZZZ")
	f.Fuzz(func(t *testing.T, amount float64, cc string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("NewMoney panicked: %v (amount=%v, currency=%q)", r, amount, cc)
			}
		}()
		m, err := money.New(amount, currency.Code(cc))
		if err == nil {
			if !currency.IsValidCurrencyFormat(string(m.Currency())) {

				t.Errorf("Money currency is invalid: %q", m.Currency())
			}
		}
	})
}
