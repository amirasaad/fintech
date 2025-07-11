package account

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// FuzzAccountDeposit tests Account.Deposit invariants with random input.
func FuzzAccountDeposit(f *testing.F) {
	userID := uuid.New()
	f.Add(100.0, "USD") // Seed input
	f.Add(-50.0, "EUR")
	f.Add(0.0, "JPY")
	f.Add(1e12, "ZZZ")
	f.Fuzz(func(t *testing.T, amount float64, cc string) {
		acc, err := New().WithUserID(userID).WithCurrency("USD").Build()
		if err != nil {
			t.Skip()
		}
		money, err := money.NewMoney(amount, currency.Code(cc))
		if err != nil {
			t.Skip()
		}
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Deposit panicked: %v (amount=%v, currency=%q)", r, amount, cc)
			}
		}()
		_, _ = acc.Deposit(userID, money)
		// Invariant: balance should never be negative
		if acc.Balance < 0 {
			t.Errorf("Account balance is negative after deposit: %d (amount=%v, currency=%q)", acc.Balance, amount, cc)
		}
		// Invariant: currency format is always valid
		if !currency.IsValidCurrencyFormat(string(acc.Currency)) {
			t.Errorf("Account currency is invalid: %q", acc.Currency)
		}
	})
}

// FuzzAccountWithdraw tests Account.Withdraw invariants with random input.
func FuzzAccountWithdraw(f *testing.F) {
	userID := uuid.New()
	f.Add(100.0, "USD") // Seed input
	f.Add(-50.0, "EUR")
	f.Add(0.0, "JPY")
	f.Add(1e6, "ZZZ")
	f.Fuzz(func(t *testing.T, amount float64, cc string) {
		acc, err := New().WithUserID(userID).WithCurrency("USD").Build()
		if err != nil {
			t.Skip()
		}
		// Deposit some funds first
		depositMoney, err := money.NewMoney(1e6, "USD")
		if err != nil {
			t.Skip()
		}
		_, _ = acc.Deposit(userID, depositMoney)
		money, err := money.NewMoney(amount, currency.Code(cc))
		if err != nil {
			t.Skip()
		}
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Withdraw panicked: %v (amount=%v, currency=%q)", r, amount, cc)
			}
		}()
		_, _ = acc.Withdraw(userID, money)
		// Invariant: balance should never be negative
		if acc.Balance < 0 {
			t.Errorf("Account balance is negative after withdraw: %d (amount=%v, currency=%q)", acc.Balance, amount, cc)
		}
		// Invariant: currency format is always valid
		// Explicitly convert string to currency.Code for validation
		if !IsValidCurrencyFormat(currency.Code(cc)) {
			t.Errorf("Account currency is invalid: %q", acc.Currency)
		}
	})
}
