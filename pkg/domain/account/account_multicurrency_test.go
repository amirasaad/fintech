package account_test

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAccount_HasCurrencyField(t *testing.T) {
	a, err := account.New().WithUserID(uuid.New()).Build()
	assert.NoError(t, err)
	assert.NotEmpty(t, a.Currency, "Account should have a Currency field")
}

func TestAccount_DefaultCurrencyIsUSD(t *testing.T) {
	a, err := account.New().WithUserID(uuid.New()).Build()
	assert.NoError(t, err)
	assert.Equal(t, currency.USD, a.Currency, "Default currency should be USD")
}

func TestAccount_CreateWithSpecificCurrency(t *testing.T) {
	a, err := account.New().WithUserID(uuid.New()).WithCurrency("EUR").Build()
	assert.NoError(t, err)
	assert.Equal(t, currency.EUR, a.Currency, "Account should use specified currency")
}

func TestTransaction_HasCurrencyField(t *testing.T) {
	tx := account.NewTransaction().WithUserID(uuid.New()).WithAccountID(uuid.New()).WithAmount(100).WithBalance(100).WithCurrency("EUR").Build()
	assert.Equal(t, currency.EUR, tx.Currency, "Transaction should have a Currency field")
}
