package domain_test

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/google/uuid"
)

// FuzzAccountDeposit tests Account.Deposit invariants with random input.
func FuzzAccountDeposit(f *testing.F) {
	userID := uuid.New()
	f.Add(100.0, "USD") // Seed input
	f.Add(-50.0, "EUR")
	f.Add(0.0, "JPY")
	f.Add(1e12, "ZZZ")
	f.Fuzz(func(t *testing.T, amount float64, currency string) {
		acc, err := domain.NewAccountWithCurrency(userID, "USD")
		if err != nil {
			t.Skip()
		}
		money, err := domain.NewMoney(amount, currency)
		if err != nil {
			t.Skip()
		}
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Deposit panicked: %v (amount=%v, currency=%q)", r, amount, currency)
			}
		}()
		_, _ = acc.Deposit(userID, money)
		// Invariant: balance should never be negative
		if acc.Balance < 0 {
			t.Errorf("Account balance is negative after deposit: %d (amount=%v, currency=%q)", acc.Balance, amount, currency)
		}
		// Invariant: currency format is always valid
		if !domain.IsValidCurrencyFormat(acc.Currency) {
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
	f.Fuzz(func(t *testing.T, amount float64, currency string) {
		acc, err := domain.NewAccountWithCurrency(userID, "USD")
		if err != nil {
			t.Skip()
		}
		// Deposit some funds first
		depositMoney, err := domain.NewMoney(1e6, "USD")
		if err != nil {
			t.Skip()
		}
		_, _ = acc.Deposit(userID, depositMoney)
		money, err := domain.NewMoney(amount, currency)
		if err != nil {
			t.Skip()
		}
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Withdraw panicked: %v (amount=%v, currency=%q)", r, amount, currency)
			}
		}()
		_, _ = acc.Withdraw(userID, money)
		// Invariant: balance should never be negative
		if acc.Balance < 0 {
			t.Errorf("Account balance is negative after withdraw: %d (amount=%v, currency=%q)", acc.Balance, amount, currency)
		}
		// Invariant: currency format is always valid
		if !domain.IsValidCurrencyFormat(acc.Currency) {
			t.Errorf("Account currency is invalid: %q", acc.Currency)
		}
	})
}
