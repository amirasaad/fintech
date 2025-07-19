package account_test

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/currency"
	domainaccount "github.com/amirasaad/fintech/pkg/domain/account"
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
		acc, err := domainaccount.New().WithUserID(userID).WithCurrency("USD").Build()
		if err != nil {
			t.Skip()
		}
		mon, err := money.New(amount, currency.Code(cc))
		if err != nil {
			t.Skip()
		}
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Deposit panicked: %v (amount=%v, currency=%q)", r, amount, cc)
			}
		}()
		_ = acc.Deposit(userID, mon, domainaccount.MoneySourceCash, "")
		// Invariant: balance should never be negative
		if notNegative, err := acc.Balance.GreaterThan(money.Zero(acc.Balance.Currency())); err != nil {
			if !notNegative {
				t.Errorf("Account balance is negative after deposit: %v (amount=%v, currency=%q)", acc.Balance, amount, cc)
			}
		}
		// Invariant: currency format is always valid
		if !currency.IsValidCurrencyFormat(string(acc.Balance.Currency())) {
			t.Errorf("Account currency is invalid: %q", acc.Balance.Currency())
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
		acc, err := domainaccount.New().WithUserID(userID).WithCurrency("USD").Build()
		if err != nil {
			t.Skip()
		}
		// Deposit some funds first
		depositMoney, err := money.New(1e6, "USD")
		if err != nil {
			t.Skip()
		}
		_ = acc.Deposit(userID, depositMoney, domainaccount.MoneySourceCash, "")
		mon, err := money.New(amount, currency.Code(cc))
		if err != nil {
			t.Skip()
		}
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Withdraw panicked: %v (amount=%v, currency=%q)", r, amount, cc)
			}
		}()
		_ = acc.ValidateWithdraw(userID, mon)
		// Invariant: balance should never be negative
		if notNegative, err := acc.Balance.GreaterThan(money.Zero(acc.Balance.Currency())); err != nil {
			if !notNegative {
				t.Errorf("Account balance is negative after deposit: %v (amount=%v, currency=%q)", acc.Balance, amount, cc)
			}
		}
		// Invariant: currency format is always valid
		// Explicitly convert string to currency.Code for validation
		if !currency.IsValidCurrencyFormat(string(acc.Balance.Currency())) {
			t.Errorf("Account currency is invalid: %q", acc.Balance.Currency())
		}
	})
}
