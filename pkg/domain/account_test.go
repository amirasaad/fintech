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
	depositTransaction, err := account.Deposit(userID, domain.Money{Amount: 100.0, Currency: "USD"})
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
	_, err := account.Deposit(userID, domain.Money{Amount: -50.0, Currency: "USD"})
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
	_, err := account.Deposit(userID, domain.Money{Amount: 0.0, Currency: "USD"})
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
	_, err1 := account.Deposit(userID, domain.Money{Amount: 50.0, Currency: "USD"})
	require.NoError(err1, "First deposit should not return an error")
	_, err2 := account.Deposit(userID, domain.Money{Amount: 150.0, Currency: "USD"})
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
	_, err := a.Deposit(userID, domain.Money{Amount: safeAmount, Currency: "USD"})
	assert.NoError(err, "Deposit amount just under max safe value should not return an error")
}

func TestDepositOverflowFails(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	userID := uuid.New()
	a := domain.NewAccount(userID)

	overflowAmount := float64(math.MaxInt64)/100.0 + 1
	_, err := a.Deposit(userID, domain.Money{Amount: overflowAmount, Currency: "USD"})
	assert.ErrorIs(err, domain.ErrDepositAmountExceedsMaxSafeInt)
}

func TestDepositOverflowBoundary(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	userID := uuid.New()
	account := domain.NewAccount(userID)
	// Deposit up to just below the max safe int
	_, err := account.Deposit(userID, domain.Money{Amount: float64(math.MaxInt64 / 200), Currency: "USD"})
	require.NoError(err, "Deposit just below overflow boundary should not return an error")

	// This deposit should cause an overflow
	_, err = account.Deposit(userID, domain.Money{Amount: float64(math.MaxInt64/200 + 1), Currency: "USD"})
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
	_, err := account.Deposit(userID, domain.Money{Amount: 99.99, Currency: "USD"})
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
	_, err := account.Deposit(userID, domain.Money{Amount: 1000000.0, Currency: "USD"}) // 1 million dollars
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
	_, err := account.Deposit(userID, domain.Money{Amount: 200.0, Currency: "USD"}) // 200 dollars
	require.NoError(err, "Initial deposit should not return an error")

	// Withdraw funds
	withdrawalAmount := 100.0 // 100 dollars
	transaction, err := account.Withdraw(userID, domain.Money{Amount: withdrawalAmount, Currency: "USD"})
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
	_, err := account.Withdraw(userID, domain.Money{Amount: 100.0, Currency: "USD"}) // 100 dollars
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
	_, err := account.Withdraw(userID, domain.Money{Amount: -50.0, Currency: "USD"})
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
	_, err := account.Withdraw(userID, domain.Money{Amount: 0.0, Currency: "USD"})
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
	_, err := account.Deposit(userID, domain.Money{Amount: 300.0, Currency: "USD"}) // 300 dollars
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
	_, err := account.Deposit(userID, domain.Money{Amount: initialBalance, Currency: "USD"})
	require.NoError(err, "Initial deposit should not return an error")

	numOperations := 1000
	depositAmount := 10.0
	withdrawAmount := 5.0

	var wg sync.WaitGroup
	wg.Add(numOperations * 2)

	for range numOperations {
		go func() {
			defer wg.Done()
			_, depositErr := account.Deposit(userID, domain.Money{Amount: depositAmount, Currency: "USD"})
			assert.NoError(depositErr, "Deposit operation should not return an error")
		}()

		go func() {
			defer wg.Done()
			_, withdrawErr := account.Withdraw(userID, domain.Money{Amount: withdrawAmount, Currency: "USD"})
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
	_, err := account.Deposit(uuid.New(), domain.Money{Amount: 1000.0, Currency: "USD"})
	assert.Error(err, "Deposit with different user id should return error")
	assert.ErrorIs(err, domain.ErrUserUnauthorized)
}

func TestAccount_WithdrawUnauthorized(t *testing.T) {
	assert := assert.New(t)
	userID := uuid.New()
	account := domain.NewAccount(userID)
	_, err := account.Withdraw(uuid.New(), domain.Money{Amount: 1000.0, Currency: "USD"})
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
