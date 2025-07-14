package account_test

import (
	"io"
	"log"
	"log/slog"
	"math"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/domain/user"
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
	assert := assert.New(t)
	acc, err := account.New().WithUserID(uuid.New()).Build()
	assert.NoError(err)
	assert.NotEmpty(acc.ID, "Account ID should not be empty")
}

func TestDeposit(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	userID := uuid.New()
	acc, err := account.New().WithUserID(userID).Build()
	require.NoError(err)
	m, err := money.New(100.0, "USD")
	require.NoError(err)
	depositTransaction, err := acc.Deposit(userID, m)
	require.NoError(err, "Deposit should not return an error")
	assert.NotNil(depositTransaction, "Deposit transaction should not be nil")
	assert.Equal(acc.ID, depositTransaction.AccountID, "Deposit transaction should reference the correct account ID")
	assert.Equal(float64(100), depositTransaction.Amount.AmountFloat(), "Deposit amount should match the expected value")
	balance, err := acc.GetBalance(userID)
	require.NoError(err, "GetBalance for same user should not return an error")
	assert.InEpsilon(100.0, balance, 0.01, "Account balance should be updated correctly after deposit")
}

func TestDepositNegativeAmount(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	userID := uuid.New()

	account, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	// Attempt to deposit a negative amount
	money, err := money.New(-50.0, "USD")
	require.NoError(err)
	_, err = account.Deposit(userID, money)
	require.Error(err, "deposit amount must be positive")
	balance, err := account.GetBalance(userID)
	require.NoError(err, "GetBalance for same user should not return an error")
	assert.InDelta(0.0, balance, 0.01, "Account balance should remain unchanged after failed deposit")
}

func TestDepositZeroAmount(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	userID := uuid.New()

	account, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	// Attempt to deposit zero amount
	money, err := money.New(0.0, "USD")
	require.NoError(err)
	_, err = account.Deposit(userID, money)
	require.Error(err, "Deposit with zero amount should return an error")
	balance, err := account.GetBalance(userID)
	require.NoError(err, "GetBalance for same user should not return an error")
	assert.InDelta(0.0, balance, 0.01, "Account balance should remain unchanged after zero deposit")
}

func TestDepositMultipleTimes(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	userID := uuid.New()
	account, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	// Deposit multiple times
	money1, err := money.New(50.0, "USD")
	require.NoError(err)
	_, err1 := account.Deposit(userID, money1)
	require.NoError(err1, "First deposit should not return an error")
	money2, err := money.New(150.0, "USD")
	require.NoError(err)
	_, err2 := account.Deposit(userID, money2)
	require.NoError(err2, "Second deposit should not return an error")

	expectedBalance := 200.0 // 50 + 150 in cents
	balance, err := account.GetBalance(userID)
	require.NoError(err, "GetBalance for same user should not return an error")
	assert.InDelta(expectedBalance, balance, 0.01, "Account balance should be updated correctly after multiple deposits")
}

func TestDepositOverflow(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	userID := uuid.New()
	a, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck

	// Use just under the max safe amount with margin
	safeAmount := float64((math.MaxInt64 - 10000) / 100) // leave margin
	money, err := money.New(safeAmount, "USD")
	require.NoError(t, err)
	_, err = a.Deposit(userID, money)
	assert.NoError(err, "Deposit amount just under max safe value should not return an error")
}

func TestDepositOverflowFails(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	userID := uuid.New()
	a, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck

	// Use an amount that can be created as Money but will cause overflow when added
	// to an account that already has a large balance
	largeAmount := float64(math.MaxInt64 / 200) // This can be created as Money
	money, err := money.New(largeAmount, "USD")
	require.NoError(t, err)

	// First deposit to get a large balance
	_, err = a.Deposit(userID, money)
	require.NoError(t, err)

	// Second deposit should cause overflow
	_, err = a.Deposit(userID, money)
	assert.ErrorIs(err, account.ErrDepositAmountExceedsMaxSafeInt)
}

func TestDepositOverflowBoundary(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	userID := uuid.New()
	acc, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	// Deposit up to just below the max safe int
	m, err := money.New(float64(math.MaxInt64/200), "USD")
	require.NoError(err)
	_, err = acc.Deposit(userID, m)
	require.NoError(err, "Deposit just below overflow boundary should not return an error")

	// This deposit should cause an overflow
	m, err = money.New(float64(math.MaxInt64/200+1), "USD")
	require.NoError(err)
	_, err = acc.Deposit(userID, m)
	require.Error(err, "Deposit that causes overflow should return an error")
	assert.Equal(account.ErrDepositAmountExceedsMaxSafeInt, err, "Error should be ErrDepositAmountExceedsMaxSafeInt")
}

func TestDepositWithPrecision(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	userID := uuid.New()
	account, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	// Deposit with precision
	money, err := money.New(99.99, "USD")
	require.NoError(err)
	_, err = account.Deposit(userID, money)
	require.NoError(err, "Deposit with precision should not return an error")
	expectedBalance := 99.99
	balance, err := account.GetBalance(userID)
	require.NoError(err, "GetBalance for same user should not return an error")
	assert.InDelta(expectedBalance, balance, 0.01, "Account balance should be updated correctly after deposit with precision")
}

func TestDepositWithLargeAmount(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	userID := uuid.New()
	account, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	// Deposit a large amount
	money, err := money.New(1000000.0, "USD") // 1 million dollars
	require.NoError(err)
	_, err = account.Deposit(userID, money)
	require.NoError(err, "Deposit with large amount should not return an error")
	expectedBalance := 1000000.0 // 1 million in cents
	balance, err := account.GetBalance(userID)
	require.NoError(err, "GetBalance for same user should not return an error")
	assert.InDelta(expectedBalance, balance, 0.01, "Account balance should be updated correctly after large deposit")
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	userID := uuid.New()
	account, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	// Deposit some funds first
	m, err := money.New(200.0, "USD") // 200 dollars
	require.NoError(err)
	_, err = account.Deposit(userID, m)
	require.NoError(err, "Initial deposit should not return an error")

	// Withdraw funds
	withdrawalAmount := 100.0 // 100 dollars
	withdrawMoney, err := money.New(withdrawalAmount, "USD")
	require.NoError(err)
	transaction, err := account.Withdraw(userID, withdrawMoney)
	require.NoError(err, "Withdrawal should not return an error")
	assert.Equal(-withdrawalAmount, transaction.Amount.AmountFloat(), "Withdrawal transaction amount should match expected")
	balance, err := account.GetBalance(userID)
	require.NoError(err, "GetBalance for same user should not return an error")
	assert.InDelta(100.0, balance, 0.01, "Account balance should be updated correctly after withdrawal")
}

func TestWithdrawInsufficientFunds(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	userID := uuid.New()

	acc, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	// Attempt to withdraw more than the balance
	m, err := money.New(100.0, "USD") // 100 dollars
	require.NoError(err)
	_, err = acc.Withdraw(userID, m)
	require.Error(err, "Withdrawal with insufficient funds should return an error")
	assert.ErrorIs(account.ErrInsufficientFunds, err, "Error message should match expected")
	balance, err := acc.GetBalance(userID)
	require.NoError(err, "GetBalance for same user should not return an error")
	assert.InDelta(0.0, balance, 0.01, "Account balance should remain unchanged after failed withdrawal")
}

func TestWithdrawNegativeAmount(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	userID := uuid.New()
	account, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	// Attempt to withdraw a negative amount
	negativeMoney, err := money.New(-50.0, "USD")
	require.NoError(err)
	_, err = account.Withdraw(userID, negativeMoney)
	require.Error(err, "Withdrawal with negative amount should return an error")
	assert.Equal("withdrawal amount must be positive", err.Error(), "Error message should match expected")
	balance, err := account.GetBalance(userID)
	require.NoError(err, "GetBalance for same user should not return an error")
	assert.InDelta(0.0, balance, 0.01, "Account balance should remain unchanged after failed withdrawal")
}

func TestWithdrawZeroAmount(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	userID := uuid.New()

	account, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	// Attempt to withdraw zero amount
	zeroMoney, err := money.New(0.0, "USD")
	require.NoError(err)
	_, err = account.Withdraw(userID, zeroMoney)
	require.Error(err, "Withdrawal with zero amount should return an error")
	balance, err := account.GetBalance(userID)
	require.NoError(err, "GetBalance for same user should not return an error")
	assert.InDelta(0.0, balance, 0.01, "Account balance should remain unchanged after zero withdrawal")
}

func TestGetBalance(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	userID := uuid.New()

	account, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	// Deposit some funds
	money, err := money.New(300.0, "USD") // 300 dollars
	require.NoError(err)
	_, err = account.Deposit(userID, money)
	require.NoError(err, "Initial deposit should not return an error")

	// Check balance
	balance, err := account.GetBalance(userID)
	require.NoError(err, "GetBalance for same user should not return an error")
	assert.InDelta(300.0, balance, 0.01, "Account balance should match the expected value after deposit")
}

func TestSimultaneous(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	userID := uuid.New()

	account, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	initialBalance := 1000.0
	m, err := money.New(initialBalance, "USD")
	require.NoError(err)
	_, err = account.Deposit(userID, m)
	require.NoError(err, "Initial deposit should not return an error")

	numOperations := 1000
	depositAmount := 10.0
	withdrawAmount := 5.0

	var wg sync.WaitGroup
	wg.Add(numOperations * 2)

	for range numOperations {
		go func() {
			defer wg.Done()
			m, errMoney := money.New(depositAmount, "USD")
			require.NoError(errMoney)
			_, depositErr := account.Deposit(userID, m)
			assert.NoError(depositErr, "Deposit operation should not return an error")
		}()

		go func() {
			defer wg.Done()
			withdrawMoney, errWithdraw := money.New(withdrawAmount, "USD")
			require.NoError(errWithdraw)
			_, withdrawErr := account.Withdraw(userID, withdrawMoney)
			assert.NoError(withdrawErr, "Withdrawal operation should not return an error")
		}()
	}

	wg.Wait()

	expectedBalance := initialBalance + (float64(numOperations) * depositAmount) - (float64(numOperations) * withdrawAmount)
	balance, err := account.GetBalance(userID)
	require.NoError(err, "GetBalance for same user should not return an error")
	assert.InDelta(expectedBalance, balance, 0.01, "Final balance should be correct after concurrent operations")
}

func TestAccount_DepositUnauthorized(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	userID := uuid.New()
	account, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	money, err := money.New(1000.0, "USD")
	require.NoError(err)
	_, err = account.Deposit(uuid.New(), money)
	require.Error(err, "Deposit with different user id should return error")
	assert.ErrorIs(err, user.ErrUserUnauthorized)
}

func TestAccount_WithdrawUnauthorized(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	userID := uuid.New()
	account, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	unauthorizedMoney, err := money.New(1000.0, "USD")
	require.NoError(err)
	_, err = account.Withdraw(uuid.New(), unauthorizedMoney)
	require.Error(err, "Deposit with different user id should return error")
	assert.ErrorIs(err, user.ErrUserUnauthorized)
}

func TestAccount_GetBalanceUnauthorized(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	userID := uuid.New()
	account, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	_, err := account.GetBalance(uuid.New())
	require.Error(err, "Deposit with different user id should return error")
	assert.ErrorIs(err, user.ErrUserUnauthorized)
}

func TestNewUserFromData(t *testing.T) {
	assert := assert.New(t)
	userID := uuid.New()
	user := user.NewUserFromData(userID, "test", "test@test.com", "password", time.Now(), time.Now())
	assert.Equal(userID, user.ID)
	assert.Equal("test", user.Username)
	assert.Equal("test@test.com", user.Email)
	assert.Equal("password", user.Password)
}

func TestNewTransactionFromData(t *testing.T) {
	assert := assert.New(t)
	userID := uuid.New()
	accountID := uuid.New()
	transactionID := uuid.New()
	amount, _ := money.New(100.0, "USD")
	balance, _ := money.New(100.0, "USD")
	transaction := account.NewTransactionFromData(transactionID, userID, accountID, amount, balance, account.MoneySourceCash, time.Now())
	assert.Equal(transactionID, transaction.ID)
	assert.Equal(userID, transaction.UserID)
	assert.Equal(accountID, transaction.AccountID)
	assert.Equal(amount, transaction.Amount)
	assert.Equal(balance, transaction.Balance)
}

func TestAccount_GetBalanceAsMoney(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	userID := uuid.New()
	account, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck

	// Initial balance should be zero
	balanceMoney, err := account.GetBalanceAsMoney(userID)
	require.NoError(err)
	assert.True(balanceMoney.IsZero())
	assert.Equal("USD", string(balanceMoney.Currency()))

	// Deposit some funds
	depositMoney, err := money.New(100.50, "USD")
	require.NoError(err)
	_, err = account.Deposit(userID, depositMoney)
	require.NoError(err)

	// Check balance as Money
	balanceMoney, err = account.GetBalanceAsMoney(userID)
	require.NoError(err)
	assert.False(balanceMoney.IsZero())
	assert.InDelta(100.50, balanceMoney.AmountFloat(), 0.001)
	assert.Equal("USD", string(balanceMoney.Currency()))

	// Verify it matches the float balance
	floatBalance, err := account.GetBalance(userID)
	require.NoError(err)
	assert.InDelta(floatBalance, balanceMoney.AmountFloat(), 0.001)
}

func TestAccount_DepositWithMoneyOperations(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	userID := uuid.New()
	account, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck

	// Deposit using Money operations
	depositMoney, err := money.New(100.0, "USD")
	require.NoError(err)
	tx, err := account.Deposit(userID, depositMoney)
	require.NoError(err)
	assert.NotNil(tx)

	// Check balance using Money
	balanceMoney, err := account.GetBalanceAsMoney(userID)
	require.NoError(err)
	assert.InDelta(100.0, balanceMoney.AmountFloat(), 0.001)

	// Deposit more using Money operations
	secondDeposit, err := money.New(50.25, "USD")
	require.NoError(err)
	tx2, err := account.Deposit(userID, secondDeposit)
	require.NoError(err)
	assert.NotNil(tx2)

	// Check final balance
	balanceMoney, err = account.GetBalanceAsMoney(userID)
	require.NoError(err)
	assert.InDelta(150.25, balanceMoney.AmountFloat(), 0.001)
}

func TestAccount_WithdrawWithMoneyOperations(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	userID := uuid.New()
	acc, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck

	// Deposit initial funds
	depositMoney, err := money.New(200.0, "USD")
	require.NoError(err)
	_, err = acc.Deposit(userID, depositMoney)
	require.NoError(err)

	// Withdraw using Money operations
	withdrawMoney, err := money.New(75.50, "USD")
	require.NoError(err)
	tx, err := acc.Withdraw(userID, withdrawMoney)
	require.NoError(err)
	assert.NotNil(tx)

	// Check balance using Money
	balanceMoney, err := acc.GetBalanceAsMoney(userID)
	require.NoError(err)
	assert.InDelta(124.50, balanceMoney.AmountFloat(), 0.001)

	// Try to withdraw more than available
	largeWithdraw, err := money.New(200.0, "USD")
	require.NoError(err)
	_, err = acc.Withdraw(userID, largeWithdraw)
	require.Error(err)
	assert.ErrorIs(err, account.ErrInsufficientFunds)

	// Balance should remain unchanged
	balanceMoney, err = acc.GetBalanceAsMoney(userID)
	require.NoError(err)
	assert.InDelta(124.50, balanceMoney.AmountFloat(), 0.001)
}
