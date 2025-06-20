package domain_test

import (
	"sync"
	"testing"

	"github.com/amirasaad/fintech/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAccount(t *testing.T) {
	assert := assert.New(t)

	// Open account should return an account ID
	account := domain.NewAccount()
	assert.NotEmpty(account.ID, "Account ID should not be empty")
}

func TestDeposit(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	account := domain.NewAccount()
	// Simulate a deposit
	depositTransaction, err := account.Deposit(100.0)
	require.NoError(err, "Deposit should not return an error")
	assert.NotNil(depositTransaction, "Deposit transaction should not be nil")
	assert.Equal(account.ID, depositTransaction.AccountID, "Deposit transaction should reference the correct account ID")
	assert.Equal(int64(100), depositTransaction.Amount/100, "Deposit amount should match the expected value")
	assert.InEpsilon(100.0, account.GetBalance(), 0.01, "Account balance should be updated correctly after deposit")
}

func TestDepositNegativeAmount(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	account := domain.NewAccount()
	// Attempt to deposit a negative amount
	_, err := account.Deposit(-50.0)
	require.Error(err, "deposit amount must be positive")
	assert.InDelta(0.0, account.GetBalance(), 0.01, "Account balance should remain unchanged after failed deposit")
}

func TestDepositZeroAmount(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	account := domain.NewAccount()
	// Attempt to deposit zero amount
	_, err := account.Deposit(0.0)
	require.NoError(err, "Deposit with zero amount should not return an error")
	assert.InDelta(0.0, account.GetBalance(), 0.01, "Account balance should remain unchanged after zero deposit")
}

func TestDepositMultipleTimes(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	account := domain.NewAccount()
	// Deposit multiple times
	_, err1 := account.Deposit(50.0)
	require.NoError(err1, "First deposit should not return an error")
	_, err2 := account.Deposit(150.0)
	require.NoError(err2, "Second deposit should not return an error")

	expectedBalance := 200.0 // 50 + 150 in cents
	assert.InDelta(expectedBalance, account.GetBalance(), 0.01, "Account balance should be updated correctly after multiple deposits")
}

func TestDepositOverflow(t *testing.T) {
	assert := assert.New(t)

	a := domain.NewAccount()
	_, err := a.Deposit(1000000000000000000000000000000000000000)
	assert.Error(err, "Deposit amount exceeds maximum safe integer value")

}
func TestDepositWithPrecision(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	account := domain.NewAccount()
	// Deposit with precision
	_, err := account.Deposit(99.99)
	require.NoError(err, "Deposit with precision should not return an error")
	expectedBalance := 99.99
	assert.InDelta(expectedBalance, account.GetBalance(), 0.01, "Account balance should be updated correctly after deposit with precision")
}

func TestDepositWithLargeAmount(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	account := domain.NewAccount()
	// Deposit a large amount
	_, err := account.Deposit(1000000.0) // 1 million dollars
	require.NoError(err, "Deposit with large amount should not return an error")
	expectedBalance := 1000000.0 // 1 million in cents
	assert.InDelta(expectedBalance, account.GetBalance(), 0.01, "Account balance should be updated correctly after large deposit")
}

func TestWithdraw(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	account := domain.NewAccount()
	// Deposit some funds first
	_, err := account.Deposit(200.0) // 200 dollars
	require.NoError(err, "Initial deposit should not return an error")

	// Withdraw funds
	withdrawalAmount := 100.0 // 100 dollars
	transaction, err := account.Withdraw(withdrawalAmount)
	require.NoError(err, "Withdrawal should not return an error")
	assert.Equal(-int64(withdrawalAmount), transaction.Amount/100, "Withdrawal transaction amount should match expected")
	assert.InDelta(100.0, account.GetBalance(), 0.01, "Account balance should be updated correctly after withdrawal")
}

func TestWithdrawInsufficientFunds(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	account := domain.NewAccount()
	// Attempt to withdraw more than the balance
	_, err := account.Withdraw(100.0) // 100 dollars
	require.Error(err, "Withdrawal with insufficient funds should return an error")
	assert.Equal("insufficient funds for withdrawal", err.Error(), "Error message should match expected")
	assert.InDelta(0.0, account.GetBalance(), 0.01, "Account balance should remain unchanged after failed withdrawal")
}

func TestWithdrawNegativeAmount(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	account := domain.NewAccount()
	// Attempt to withdraw a negative amount
	_, err := account.Withdraw(-50.0)
	require.Error(err, "Withdrawal with negative amount should return an error")
	assert.Equal("withdrawal amount must be positive", err.Error(), "Error message should match expected")
	assert.InDelta(0.0, account.GetBalance(), 0.01, "Account balance should remain unchanged after failed withdrawal")
}

func TestWithdrawZeroAmount(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	account := domain.NewAccount()
	// Attempt to withdraw zero amount
	_, err := account.Withdraw(0.0)
	require.NoError(err, "Withdrawal with zero amount should not return an error")
	assert.InDelta(0.0, account.GetBalance(), 0.01, "Account balance should remain unchanged after zero withdrawal")
}

func TestGetBalance(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	account := domain.NewAccount()
	// Deposit some funds
	_, err := account.Deposit(300.0) // 300 dollars
	require.NoError(err, "Initial deposit should not return an error")

	// Check balance
	balance := account.GetBalance()
	assert.InDelta(300.0, balance, 0.01, "Account balance should match the expected value after deposit")
}

func TestSimultaneous(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	account := domain.NewAccount()
	initialBalance := 1000.0
	_, err := account.Deposit(initialBalance)
	require.NoError(err, "Initial deposit should not return an error")

	numOperations := 1000
	depositAmount := 10.0
	withdrawAmount := 5.0

	var wg sync.WaitGroup
	wg.Add(numOperations * 2)

	for range numOperations {
		go func() {
			defer wg.Done()
			_, err := account.Deposit(depositAmount)
			assert.NoError(err, "Deposit operation should not return an error")
		}()

		go func() {
			defer wg.Done()
			_, err := account.Withdraw(withdrawAmount)
			assert.NoError(err, "Withdrawal operation should not return an error")
		}()
	}

	wg.Wait()

	expectedBalance := initialBalance + (float64(numOperations) * depositAmount) - (float64(numOperations) * withdrawAmount)
	assert.InDelta(expectedBalance, account.GetBalance(), 0.01, "Final balance should be correct after concurrent operations")
}
