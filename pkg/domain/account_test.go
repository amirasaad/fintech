package domain_test

import (
	"io"
	"log"
	"log/slog"
	"math"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/amirasaad/fintech/pkg/domain"
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

	// Open account should return an account ID
	account := domain.NewAccount(uuid.New())
	assert.NotEmpty(account.ID, "Account ID should not be empty")
}

func TestDeposit(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	userID := uuid.New()
	account := domain.NewAccount(userID)
	// Simulate a deposit
	money, err := domain.NewMoney(100.0, "USD")
	require.NoError(err)
	depositTransaction, err := account.Deposit(userID, money)
	require.NoError(err, "Deposit should not return an error")
	assert.NotNil(depositTransaction, "Deposit transaction should not be nil")
	assert.Equal(account.ID, depositTransaction.AccountID, "Deposit transaction should reference the correct account ID")
	assert.Equal(int64(100), depositTransaction.Amount/100, "Deposit amount should match the expected value")
	balance, err := account.GetBalance(userID)
	require.NoError(err, "GetBalance for same user should not return an error")
	assert.InEpsilon(100.0, balance, 0.01, "Account balance should be updated correctly after deposit")
}

func TestDepositNegativeAmount(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	userID := uuid.New()

	account := domain.NewAccount(userID)
	// Attempt to deposit a negative amount
	money, err := domain.NewMoney(-50.0, "USD")
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

	account := domain.NewAccount(userID)
	// Attempt to deposit zero amount
	money, err := domain.NewMoney(0.0, "USD")
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
	account := domain.NewAccount(userID)
	// Deposit multiple times
	money1, err := domain.NewMoney(50.0, "USD")
	require.NoError(err)
	_, err1 := account.Deposit(userID, money1)
	require.NoError(err1, "First deposit should not return an error")
	money2, err := domain.NewMoney(150.0, "USD")
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
	a := domain.NewAccount(userID)

	// Use just under the max safe amount with margin
	safeAmount := float64((math.MaxInt64 - 10000) / 100) // leave margin
	money, err := domain.NewMoney(safeAmount, "USD")
	require.NoError(t, err)
	_, err = a.Deposit(userID, money)
	assert.NoError(err, "Deposit amount just under max safe value should not return an error")
}

func TestDepositOverflowFails(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	userID := uuid.New()
	a := domain.NewAccount(userID)

	// Use an amount that can be created as Money but will cause overflow when added
	// to an account that already has a large balance
	largeAmount := float64(math.MaxInt64 / 200) // This can be created as Money
	money, err := domain.NewMoney(largeAmount, "USD")
	require.NoError(t, err)

	// First deposit to get a large balance
	_, err = a.Deposit(userID, money)
	require.NoError(t, err)

	// Second deposit should cause overflow
	_, err = a.Deposit(userID, money)
	assert.ErrorIs(err, domain.ErrDepositAmountExceedsMaxSafeInt)
}

func TestDepositOverflowBoundary(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	userID := uuid.New()
	account := domain.NewAccount(userID)
	// Deposit up to just below the max safe int
	money, err := domain.NewMoney(float64(math.MaxInt64/200), "USD")
	require.NoError(err)
	_, err = account.Deposit(userID, money)
	require.NoError(err, "Deposit just below overflow boundary should not return an error")

	// This deposit should cause an overflow
	money, err = domain.NewMoney(float64(math.MaxInt64/200+1), "USD")
	require.NoError(err)
	_, err = account.Deposit(userID, money)
	require.Error(err, "Deposit that causes overflow should return an error")
	assert.Equal(domain.ErrDepositAmountExceedsMaxSafeInt, err, "Error should be ErrDepositAmountExceedsMaxSafeInt")
}

func TestDepositWithPrecision(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	userID := uuid.New()
	account := domain.NewAccount(userID)
	// Deposit with precision
	money, err := domain.NewMoney(99.99, "USD")
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
	account := domain.NewAccount(userID)
	// Deposit a large amount
	money, err := domain.NewMoney(1000000.0, "USD") // 1 million dollars
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
	account := domain.NewAccount(userID)
	// Deposit some funds first
	money, err := domain.NewMoney(200.0, "USD") // 200 dollars
	require.NoError(err)
	_, err = account.Deposit(userID, money)
	require.NoError(err, "Initial deposit should not return an error")

	// Withdraw funds
	withdrawalAmount := 100.0 // 100 dollars
	withdrawMoney, err := domain.NewMoney(withdrawalAmount, "USD")
	require.NoError(err)
	transaction, err := account.Withdraw(userID, withdrawMoney)
	require.NoError(err, "Withdrawal should not return an error")
	assert.Equal(-int64(withdrawalAmount), transaction.Amount/100, "Withdrawal transaction amount should match expected")
	balance, err := account.GetBalance(userID)
	require.NoError(err, "GetBalance for same user should not return an error")
	assert.InDelta(100.0, balance, 0.01, "Account balance should be updated correctly after withdrawal")
}

func TestWithdrawInsufficientFunds(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	userID := uuid.New()

	account := domain.NewAccount(userID)
	// Attempt to withdraw more than the balance
	money, err := domain.NewMoney(100.0, "USD") // 100 dollars
	require.NoError(err)
	_, err = account.Withdraw(userID, money)
	require.Error(err, "Withdrawal with insufficient funds should return an error")
	assert.ErrorIs(domain.ErrInsufficientFunds, err, "Error message should match expected")
	balance, err := account.GetBalance(userID)
	require.NoError(err, "GetBalance for same user should not return an error")
	assert.InDelta(0.0, balance, 0.01, "Account balance should remain unchanged after failed withdrawal")
}

func TestWithdrawNegativeAmount(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	userID := uuid.New()
	account := domain.NewAccount(userID)
	// Attempt to withdraw a negative amount
	negativeMoney, err := domain.NewMoney(-50.0, "USD")
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

	account := domain.NewAccount(userID)
	// Attempt to withdraw zero amount
	zeroMoney, err := domain.NewMoney(0.0, "USD")
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

	account := domain.NewAccount(userID)
	// Deposit some funds
	money, err := domain.NewMoney(300.0, "USD") // 300 dollars
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

	account := domain.NewAccount(userID)
	initialBalance := 1000.0
	money, err := domain.NewMoney(initialBalance, "USD")
	require.NoError(err)
	_, err = account.Deposit(userID, money)
	require.NoError(err, "Initial deposit should not return an error")

	numOperations := 1000
	depositAmount := 10.0
	withdrawAmount := 5.0

	var wg sync.WaitGroup
	wg.Add(numOperations * 2)

	for range numOperations {
		go func() {
			defer wg.Done()
			money, err := domain.NewMoney(depositAmount, "USD")
			require.NoError(err)
			_, depositErr := account.Deposit(userID, money)
			assert.NoError(depositErr, "Deposit operation should not return an error")
		}()

		go func() {
			defer wg.Done()
			withdrawMoney, err := domain.NewMoney(withdrawAmount, "USD")
			require.NoError(err)
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
	userID := uuid.New()
	account := domain.NewAccount(userID)
	money, err := domain.NewMoney(1000.0, "USD")
	require.NoError(t, err)
	_, err = account.Deposit(uuid.New(), money)
	assert.Error(err, "Deposit with different user id should return error")
	assert.ErrorIs(err, domain.ErrUserUnauthorized)
}

func TestAccount_WithdrawUnauthorized(t *testing.T) {
	assert := assert.New(t)
	userID := uuid.New()
	account := domain.NewAccount(userID)
	unauthorizedMoney, err := domain.NewMoney(1000.0, "USD")
	require.NoError(t, err)
	_, err = account.Withdraw(uuid.New(), unauthorizedMoney)
	assert.Error(err, "Deposit with different user id should return error")
	assert.ErrorIs(err, domain.ErrUserUnauthorized)
}

func TestAccount_GetBalanceUnauthorized(t *testing.T) {
	assert := assert.New(t)
	userID := uuid.New()
	account := domain.NewAccount(userID)
	_, err := account.GetBalance(uuid.New())
	assert.Error(err, "Deposit with different user id should return error")
	assert.ErrorIs(err, domain.ErrUserUnauthorized)
}

func TestNewUserFromData(t *testing.T) {
	assert := assert.New(t)
	userID := uuid.New()
	user := domain.NewUserFromData(userID, "test", "test@test.com", "password", time.Now(), time.Now())
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
	transaction := domain.NewTransactionFromData(transactionID, userID, accountID, 100, 100, "USD", time.Now(), nil, nil, nil)
	assert.Equal(transactionID, transaction.ID)
	assert.Equal(userID, transaction.UserID)
	assert.Equal(accountID, transaction.AccountID)
	assert.Equal(int64(100), transaction.Amount)
	assert.Equal(int64(100), transaction.Balance)
}

func TestAccount_GetBalanceAsMoney(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	userID := uuid.New()
	account := domain.NewAccount(userID)

	// Initial balance should be zero
	balanceMoney, err := account.GetBalanceAsMoney(userID)
	require.NoError(err)
	assert.True(balanceMoney.IsZero())
	assert.Equal("USD", string(balanceMoney.Currency()))

	// Deposit some funds
	depositMoney, err := domain.NewMoney(100.50, "USD")
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
	account := domain.NewAccount(userID)

	// Deposit using Money operations
	depositMoney, err := domain.NewMoney(100.0, "USD")
	require.NoError(err)
	tx, err := account.Deposit(userID, depositMoney)
	require.NoError(err)
	assert.NotNil(tx)

	// Check balance using Money
	balanceMoney, err := account.GetBalanceAsMoney(userID)
	require.NoError(err)
	assert.InDelta(100.0, balanceMoney.AmountFloat(), 0.001)

	// Deposit more using Money operations
	secondDeposit, err := domain.NewMoney(50.25, "USD")
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
	account := domain.NewAccount(userID)

	// Deposit initial funds
	depositMoney, err := domain.NewMoney(200.0, "USD")
	require.NoError(err)
	_, err = account.Deposit(userID, depositMoney)
	require.NoError(err)

	// Withdraw using Money operations
	withdrawMoney, err := domain.NewMoney(75.50, "USD")
	require.NoError(err)
	tx, err := account.Withdraw(userID, withdrawMoney)
	require.NoError(err)
	assert.NotNil(tx)

	// Check balance using Money
	balanceMoney, err := account.GetBalanceAsMoney(userID)
	require.NoError(err)
	assert.InDelta(124.50, balanceMoney.AmountFloat(), 0.001)

	// Try to withdraw more than available
	largeWithdraw, err := domain.NewMoney(200.0, "USD")
	require.NoError(err)
	_, err = account.Withdraw(userID, largeWithdraw)
	assert.Error(err)
	assert.ErrorIs(err, domain.ErrInsufficientFunds)

	// Balance should remain unchanged
	balanceMoney, err = account.GetBalanceAsMoney(userID)
	require.NoError(err)
	assert.InDelta(124.50, balanceMoney.AmountFloat(), 0.001)
}
