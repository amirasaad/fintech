package account_test

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccount_DefaultCurrencyIsUSD(t *testing.T) {
	a, err := account.New().WithUserID(uuid.New()).Build()
	require.NoError(t, err)
	assert.Equal(t, currency.USD, a.Balance.Currency(), "Default currency should be USD")
}

func TestAccount_CreateWithSpecificCurrency(t *testing.T) {
	a, err := account.New().WithUserID(uuid.New()).WithCurrency("EUR").Build()
	require.NoError(t, err)
	assert.Equal(t, currency.EUR, a.Balance.Currency(), "Account should use specified currency")
}
