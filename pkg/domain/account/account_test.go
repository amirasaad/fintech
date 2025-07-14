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
	err = acc.Deposit(userID, money, domainaccount.MoneySourceInternal)
	require.Error(err, "deposit amount must be positive")
	events := acc.PullEvents()
	require.Len(events, 0)
}

func TestDepositZeroAmount(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	userID := uuid.New()
	acc, _ := domainaccount.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	money, err := money.New(0.0, "USD")
	require.NoError(err)
	err = acc.Deposit(userID, money, domainaccount.MoneySourceInternal)
	require.Error(err, "Deposit with zero amount should return an error")
	events := acc.PullEvents()
	require.Len(events, 0)
}

func TestAccount_DepositUnauthorized(t *testing.T) {
	require := require.New(t)
	userID := uuid.New()
	acc, _ := domainaccount.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	money, err := money.New(1000.0, "USD")
	require.NoError(err)
	err = acc.Deposit(uuid.New(), money, domainaccount.MoneySourceCash)
	require.Error(err, "Deposit with different user id should return error")
	events := acc.PullEvents()
	require.Len(events, 0)
}

func TestAccount_WithdrawUnauthorized(t *testing.T) {
	require := require.New(t)
	userID := uuid.New()
	acc, _ := domainaccount.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	unauthorizedMoney, err := money.New(1000.0, "USD")
	require.NoError(err)
	err = acc.Withdraw(uuid.New(), unauthorizedMoney, domainaccount.MoneySourceCash)
	require.Error(err, "Withdraw with different user id should return error")
	events := acc.PullEvents()
	require.Len(events, 0)
}

func TestAccount_GetBalanceUnauthorized(t *testing.T) {
	require := require.New(t)
	userID := uuid.New()
	acc, _ := domainaccount.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	_, err := acc.GetBalance(uuid.New())
	require.Error(err, "GetBalance with different user id should return error")
}

func TestAccount_DepositWithMoneyOperations(t *testing.T) {
	t.Parallel()
	require := require.New(t)

	userID := uuid.New()
	account, _ := domainaccount.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck

	// Deposit using Money operations
	depositMoney := money.NewFromData(100.0, "USD")
	err := account.Deposit(userID, depositMoney, domainaccount.MoneySourceCash)
	require.NoError(err)

	// Check balance using Money
	balanceMoney, err := account.GetBalanceAsMoney(userID)
	require.NoError(err)
	assert.InEpsilon(t, 100.0, balanceMoney.AmountFloat(), 0.001)

	// Deposit more using Money operations
	secondDeposit, err := money.New(50.25, "USD")
	require.NoError(err)
	err = account.Deposit(userID, secondDeposit, domainaccount.MoneySourceCash)
	require.NoError(err)

	// Check final balance
	balanceMoney, err = account.GetBalanceAsMoney(userID)
	require.NoError(err)
	assert.InEpsilon(t, 150.25, balanceMoney.AmountFloat(), 0.001)
}

func TestAccount_WithdrawWithMoneyOperations(t *testing.T) {
	t.Parallel()
	require := require.New(t)

	userID := uuid.New()
	acc, _ := domainaccount.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck

	// Deposit initial funds
	depositMoney, err := money.New(200.0, "USD")
	require.NoError(err)
	err = acc.Deposit(userID, depositMoney, domainaccount.MoneySourceCash)
	require.NoError(err)

	// Withdraw using Money operations
	withdrawMoney, err := money.New(75.50, "USD")
	require.NoError(err)
	err = acc.Withdraw(userID, withdrawMoney, domainaccount.MoneySourceCash)
	require.NoError(err)

	// Check balance using Money
	balanceMoney, err := acc.GetBalanceAsMoney(userID)
	require.NoError(err)
	assert.InEpsilon(t, 124.50, balanceMoney.AmountFloat(), 0.001)

	// Try to withdraw more than available
	largeWithdraw, err := money.New(200.0, "USD")
	require.NoError(err)
	err = acc.Withdraw(userID, largeWithdraw, domainaccount.MoneySourceCash)
	require.Error(err)
	require.ErrorIs(err, domainaccount.ErrInsufficientFunds)

	// Balance should remain unchanged
	balanceMoney, err = acc.GetBalanceAsMoney(userID)
	require.NoError(err)
	assert.InEpsilon(t, 124.50, balanceMoney.AmountFloat(), 0.001)
}

func TestDeposit_EmitsEvent(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	userID := uuid.New()
	acc, err := domainaccount.New().WithUserID(userID).Build()
	require.NoError(err)
	m, err := money.New(100.0, "USD")
	require.NoError(err)
	err = acc.Deposit(userID, m, domainaccount.MoneySourceCash)
	require.NoError(err)
	events := acc.PullEvents()
	require.Len(events, 1)
	evt, ok := events[0].(domainaccount.DepositRequestedEvent)
	require.True(ok)
	assert.Equal(t, acc.ID.String(), evt.AccountID)
	assert.Equal(t, userID.String(), evt.UserID)
	assert.InEpsilon(t, 100.0, evt.Amount, 0.01)
	assert.Equal(t, "USD", evt.Currency)
	assert.Equal(t, domainaccount.MoneySourceCash, evt.Source)
}

func TestWithdraw_EmitsEvent(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	userID := uuid.New()
	acc, err := domainaccount.New().WithUserID(userID).Build()
	require.NoError(err)
	m, err := money.New(50.0, "USD")
	require.NoError(err)
	err = acc.Withdraw(userID, m, domainaccount.MoneySourceCash)
	require.NoError(err)
	events := acc.PullEvents()
	require.Len(events, 1)
	evt, ok := events[0].(domainaccount.WithdrawRequestedEvent)
	require.True(ok)
	assert.Equal(t, acc.ID.String(), evt.AccountID)
	assert.Equal(t, userID.String(), evt.UserID)
	assert.InEpsilon(t, 50.0, evt.Amount, 0.01)
	assert.Equal(t, "USD", evt.Currency)
	assert.Equal(t, domainaccount.MoneySourceCash, evt.Source)
}

func TestTransfer_EmitsEvent(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	userID := uuid.New()
	source, _ := domainaccount.New().WithUserID(userID).WithCurrency(currency.USD).Build()
	dest, _ := domainaccount.New().WithUserID(uuid.New()).WithCurrency(currency.USD).Build()
	m, err := money.New(25.0, "USD")
	require.NoError(err)
	err = source.Transfer(userID, dest, m, domainaccount.MoneySourceInternal)
	require.NoError(err)
	events := source.PullEvents()
	require.Len(events, 1)
	evt, ok := events[0].(domainaccount.TransferRequestedEvent)
	require.True(ok)
	assert.Equal(t, source.ID.String(), evt.SourceAccountID)
	assert.Equal(t, dest.ID.String(), evt.DestAccountID)
	assert.Equal(t, userID.String(), evt.UserID)
	assert.InEpsilon(t, 25.0, evt.Amount, 0.01)
	assert.Equal(t, "USD", evt.Currency)
	assert.Equal(t, domainaccount.MoneySourceInternal, evt.Source)
}
