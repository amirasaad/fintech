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
	a, err := domain.NewAccountWithCurrency(uuid.New(), "EUR")
	assert.NoError(t, err)
	assert.Equal(t, "EUR", a.Currency, "Account should use specified currency")
}

func TestTransaction_HasCurrencyField(t *testing.T) {
	tx := domain.NewTransactionWithCurrency(uuid.New(), uuid.New(), uuid.New(), 100, 100, "EUR")
	assert.Equal(t, "EUR", tx.Currency, "Transaction should have a Currency field")
}
