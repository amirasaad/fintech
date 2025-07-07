package domain_test

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAccount_HasCurrencyField(t *testing.T) {
	a := domain.NewAccount(uuid.New())
	// This will fail until the Account struct has a Currency field
	assert.NotEmpty(t, a.Currency, "Account should have a Currency field")
}

func TestAccount_DefaultCurrencyIsUSD(t *testing.T) {
	a := domain.NewAccount(uuid.New())
	assert.Equal(t, "USD", a.Currency, "Default currency should be USD")
}

func TestAccount_CreateWithSpecificCurrency(t *testing.T) {
	a := domain.NewAccountWithCurrency(uuid.New(), "EUR")
	assert.Equal(t, "EUR", a.Currency, "Account should use specified currency")
}

func TestTransaction_HasCurrencyField(t *testing.T) {
	tx := domain.NewTransactionWithCurrency(uuid.New(), uuid.New(), uuid.New(), 100, 100, "EUR")
	assert.Equal(t, "EUR", tx.Currency, "Transaction should have a Currency field")
}

func TestPreventMixingCurrencies(t *testing.T) {
	a := domain.NewAccountWithCurrency(uuid.New(), "USD")
	// Simulate a deposit with mismatched currency
	_, err := a.DepositWithCurrency(uuid.New(), 100, "EUR")
	assert.Error(t, err, "Should not allow deposit with mismatched currency")
}

func TestValidateISOCurrencyCode(t *testing.T) {
	valid := domain.IsValidCurrencyCode("USD")
	invalid := domain.IsValidCurrencyCode("ZZZ")
	assert.True(t, valid, "USD should be a valid ISO 4217 code")
	assert.False(t, invalid, "ZZZ should not be a valid ISO 4217 code")
}
