package domain_test

import (
	"sync"
	"testing"

	"github.com/amirasaad/fintech/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestNewAccount(t *testing.T) {
	assert := assert.New(t)

	// Open account should return an account ID
	account := domain.NewAccount()
	assert.NotEmpty(account.ID, "Account ID should not be empty")
}

func TestDeposit(t *testing.T) {
	assert := assert.New(t)

	account := domain.NewAccount()
	// Simulate a deposit
	depositTransaction, err := account.Deposit(100.0)
	assert.NoError(err, "Deposit should not return an error")
	assert.NotNil(depositTransaction, "Deposit transaction should not be nil")
	assert.Equal(depositTransaction.AccountID, account.ID, "Deposit transaction should reference the correct account ID")
	assert.Equal(depositTransaction.Amount/100, int64(100), "Deposit amount should match the expected value")
	assert.Equal(account.GetBalance(), 100.0, "Account balance should be updated correctly after deposit")
}

func TestDepositNegativeAmount(t *testing.T) {
	assert := assert.New(t)

	account := domain.NewAccount()
	// Attempt to deposit a negative amount
	_, err := account.Deposit(-50.0)
	assert.Error(err, "Deposit with negative amount should return an error")
	assert.Equal(err.Error(), "Deposit amount must be positive", "Error message should match expected")
	assert.Equal(account.GetBalance(), 0.0, "Account balance should remain unchanged after failed deposit")
}

func TestDepositZeroAmount(t *testing.T) {
	assert := assert.New(t)

	account := domain.NewAccount()
	// Attempt to deposit zero amount
	_, err := account.Deposit(0.0)
	assert.NoError(err, "Deposit with zero amount should not return an error")
	assert.Equal(account.GetBalance(), 0.0, "Account balance should remain unchanged after zero deposit")
}

func TestDepositMultipleTimes(t *testing.T) {
	assert := assert.New(t)

	account := domain.NewAccount()
	// Deposit multiple times
	_, err1 := account.Deposit(50.0)
	assert.NoError(err1, "First deposit should not return an error")
	_, err2 := account.Deposit(150.0)
	assert.NoError(err2, "Second deposit should not return an error")

	expectedBalance := 200.0 // 50 + 150 in cents
	assert.Equal(account.GetBalance(), expectedBalance, "Account balance should be updated correctly after multiple deposits")
}

func TestDepositOverflow(t *testing.T) {
	assert := assert.New(t)

	a := domain.NewAccount()
	_, err := a.Deposit(1000000000000000000000000000000000000000)
	assert.Error(err, "Deposit amount exceeds maximum safe integer value")

}
func TestDepositWithPrecision(t *testing.T) {
	assert := assert.New(t)

	account := domain.NewAccount()
	// Deposit with precision
	_, err := account.Deposit(99.99)
	assert.NoError(err, "Deposit with precision should not return an error")
	expectedBalance := 99.99
	assert.Equal(account.GetBalance(), expectedBalance, "Account balance should be updated correctly after deposit with precision")
}

func TestDepositWithLargeAmount(t *testing.T) {
	assert := assert.New(t)

	account := domain.NewAccount()
	// Deposit a large amount
	_, err := account.Deposit(1000000.0) // 1 million dollars
	assert.NoError(err, "Deposit with large amount should not return an error")
	expectedBalance := 1000000.0 // 1 million in cents
	assert.Equal(account.GetBalance(), expectedBalance, "Account balance should be updated correctly after large deposit")
}

func TestWithdraw(t *testing.T) {
	assert := assert.New(t)

	account := domain.NewAccount()
	// Deposit some funds first
	_, err := account.Deposit(200.0) // 200 dollars
	assert.NoError(err, "Initial deposit should not return an error")

	// Withdraw funds
	withdrawalAmount := 100.0 // 100 dollars
	transaction, err := account.Withdraw(withdrawalAmount)
	assert.NoError(err, "Withdrawal should not return an error")
	assert.Equal(transaction.Amount/100, -int64(withdrawalAmount), "Withdrawal transaction amount should match expected")
	assert.Equal(account.GetBalance(), 100.0, "Account balance should be updated correctly after withdrawal")
}

func TestWithdrawInsufficientFunds(t *testing.T) {
	assert := assert.New(t)

	account := domain.NewAccount()
	// Attempt to withdraw more than the balance
	_, err := account.Withdraw(100.0) // 100 dollars
	assert.Error(err, "Withdrawal with insufficient funds should return an error")
	assert.Equal(err.Error(), "Insufficient funds for withdrawal", "Error message should match expected")
	assert.Equal(account.GetBalance(), 0.0, "Account balance should remain unchanged after failed withdrawal")
}

func TestWithdrawNegativeAmount(t *testing.T) {
	assert := assert.New(t)

	account := domain.NewAccount()
	// Attempt to withdraw a negative amount
	_, err := account.Withdraw(-50.0)
	assert.Error(err, "Withdrawal with negative amount should return an error")
	assert.Equal(err.Error(), "Withdrawal amount must be positive", "Error message should match expected")
	assert.Equal(account.GetBalance(), 0.0, "Account balance should remain unchanged after failed withdrawal")
}

func TestWithdrawZeroAmount(t *testing.T) {
	assert := assert.New(t)

	account := domain.NewAccount()
	// Attempt to withdraw zero amount
	_, err := account.Withdraw(0.0)
	assert.NoError(err, "Withdrawal with zero amount should not return an error")
	assert.Equal(account.GetBalance(), 0.0, "Account balance should remain unchanged after zero withdrawal")
}

func TestGetBalance(t *testing.T) {
	assert := assert.New(t)

	account := domain.NewAccount()
	// Deposit some funds
	_, err := account.Deposit(300.0) // 300 dollars
	assert.NoError(err, "Initial deposit should not return an error")

	// Check balance
	balance := account.GetBalance()
	assert.Equal(balance, 300.0, "Account balance should match the expected value after deposit")
}


func TestSimultaneous(t *testing.T) {
	assert := assert.New(t)

	account := domain.NewAccount()
	initialBalance := 1000.0
	_, err := account.Deposit(initialBalance)
	assert.NoError(err, "Initial deposit should not return an error")

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
	assert.Equal(expectedBalance, account.GetBalance(), "Final balance should be correct after concurrent operations")
}