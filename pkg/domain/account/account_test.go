package account_test

import (
	"io"
	"log"
	"log/slog"
	"os"
	"testing"

	"github.com/amirasaad/fintech/pkg/currency"
	domainaccount "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain runs before any tests and applies globally for all tests in the package.
func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	log.SetOutput(io.Discard)

	exitVal := m.Run()
	os.Exit(exitVal)
}
func TestNewAccount(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	acc, err := domainaccount.New().WithUserID(uuid.New()).Build()
	require.NoError(err)
	assert.NotEmpty(t, acc.ID, "Account ID should not be empty")
}

func TestDepositNegativeAmount(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	userID := uuid.New()
	acc, _ := domainaccount.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	money, err := money.New(-50.0, "USD")
	require.NoError(err)
	err = acc.Deposit(userID, money, domainaccount.MoneySourceInternal, "")
	require.Error(err, "deposit amount must be positive")
	events := acc.PullEvents()
	require.Empty(events)
}

func TestDepositZeroAmount(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	userID := uuid.New()
	acc, _ := domainaccount.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	money, err := money.New(0.0, "USD")
	require.NoError(err)
	err = acc.Deposit(userID, money, domainaccount.MoneySourceInternal, "")
	require.Error(err, "Deposit with zero amount should return an error")
	events := acc.PullEvents()
	require.Empty(events)
}

func TestAccount_DepositUnauthorized(t *testing.T) {
	require := require.New(t)
	userID := uuid.New()
	acc, _ := domainaccount.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	money, err := money.New(1000.0, "USD")
	require.NoError(err)
	err = acc.Deposit(uuid.New(), money, domainaccount.MoneySourceCash, "")
	require.Error(err, "Deposit with different user id should return error")
	events := acc.PullEvents()
	require.Empty(events)
}

func TestAccount_WithdrawUnauthorized(t *testing.T) {
	require := require.New(t)
	userID := uuid.New()
	acc, _ := domainaccount.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	amount, err := money.New(1000.0, "USD")
	require.NoError(err)
	err = acc.ValidateWithdraw(uuid.New(), amount)

	require.Error(err, "Withdraw with different user id should return error")
	events := acc.PullEvents()
	require.Empty(events)
}

func TestAccount_GetBalanceUnauthorized(t *testing.T) {
	require := require.New(t)
	userID := uuid.New()
	acc, _ := domainaccount.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	_, err := acc.GetBalance(uuid.New())
	require.Error(err, "GetBalance with different user id should return error")
}

func TestDeposit_EmitsEvent(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	userID := uuid.New()
	acc, err := domainaccount.New().WithUserID(userID).Build()
	require.NoError(err)
	m, err := money.New(100.0, "USD")
	require.NoError(err)
	err = acc.Deposit(userID, m, domainaccount.MoneySourceCash, "")
	require.NoError(err)
}

func TestWithdraw_EmitsEvent(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	userID := uuid.New()
	// Build account with sufficient initial balance
	acc, err := domainaccount.New().WithUserID(userID).WithBalance(10000).Build() // 100.00 USD in cents
	require.NoError(err)
	m, err := money.New(50.0, "USD")
	require.NoError(err)
	err = acc.ValidateWithdraw(userID, m)
	require.NoError(err)
}

func TestTransfer_EmitsEvent(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	userID := uuid.New()
	source, _ := domainaccount.New().WithUserID(userID).WithBalance(25.0).WithCurrency(currency.USD).Build()
	dest, _ := domainaccount.New().WithUserID(uuid.New()).WithCurrency(currency.USD).Build()
	m := money.NewFromData(25.0, "USD")
	err := source.Transfer(userID, uuid.New(), dest, m, domainaccount.MoneySourceInternal)
	require.NoError(err)
}
