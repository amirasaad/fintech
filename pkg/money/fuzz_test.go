package money_test

import (
	"github.com/amirasaad/fintech/pkg/money"
	"testing"
)

// FuzzNewMoney tests NewMoney invariants with random input.
func FuzzNewMoney(f *testing.F) {
	// Only add valid currency codes as seed values
	f.Add(100.0, "USD")
	f.Add(-50.0, "EUR")
	f.Add(0.0, "JPY")
	f.Add(1e12, "KWD")
	f.Add(100.0, "GBP")

	f.Fuzz(func(t *testing.T, amount float64, cc string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("NewMoney panicked: %v (amount=%v, currency=%q)", r, amount, cc)
			}
		}()

		// Skip invalid currency codes in fuzzing
		currency := money.Code(cc)
		if !currency.IsValid() {
			t.Skip("Skipping invalid currency code")
		}

		m, err := money.New(amount, currency)
		if err != nil {
			t.Fatalf("Failed to create money: %v", err)
		}

		// Verify currency code is preserved
		if got := m.CurrencyCode(); got != currency {
			t.Errorf("Currency code changed: got %q, want %q", got, currency)
		}

		// Verify basic amount handling
		switch {
		case amount > 0 && m.Amount() <= 0:
			t.Errorf("Positive amount %f became non-positive: %d", amount, m.Amount())
		case amount < 0 && m.Amount() >= 0:
			t.Errorf("Negative amount %f became non-negative: %d", amount, m.Amount())
		case amount == 0 && m.Amount() != 0:
			t.Errorf("Zero amount became non-zero: %d", m.Amount())
		}
	})
}
