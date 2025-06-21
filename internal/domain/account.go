package domain

import (
	"errors"
	"fmt"
	"time"
	"sync"

	"github.com/google/uuid"
)

type Account struct{
	ID      uuid.UUID
	Balance int64
	Created time.Time
	Updated time.Time
	mu      sync.Mutex
}

type Transaction struct {
	ID        uuid.UUID
	AccountID uuid.UUID
	Amount    int64
	Created   time.Time
}

func NewAccount() *Account {
	return &Account{
		ID:      uuid.New(),
		Created: time.Now(),
		Updated: time.Now(),
		Balance: 0,
		mu:      sync.Mutex{},
	}
}

func NewAccountFromData(id uuid.UUID, balance int64, created, updated time.Time) *Account {
	return &Account{
		ID:      id,
		Balance: balance,
		Created: created,
		Updated: updated,
		mu:      sync.Mutex{},
	}
}

func NewTransactionFromData(id, accountID uuid.UUID, amount int64, created time.Time) *Transaction {
	return &Transaction{
		ID:        id,
		AccountID: accountID,
		Amount:    amount,
		Created:   created,
	}
}

// Deposit adds funds to the account and returns a transaction record.
// The amount is expected to be in dollars, and it will be converted to cents for precision.
// It returns an error if the deposit amount is negative.
func (a *Account) Deposit(amount float64) (*Transaction, error) {
	fmt.Println("Balance before deposit:", a.Balance)
	a.mu.Lock()
	defer a.mu.Unlock()
	// Check if the amount is positive before proceeding with the deposit
	if amount < 0 {
		return nil, errors.New("Deposit amount must be positive")
	}

	parsedAmount := int64(amount * 100) // Convert to cents for precision
	if parsedAmount+a.Balance < 0 {
		return nil, errors.New("Deposit amount exceeds maximum safe integer value")
	}
	fmt.Println("Depositing amount:", amount)
	transaction := Transaction{
		ID:        uuid.New(),
		AccountID: a.ID,
		Amount:    parsedAmount,
		Created:   time.Now().UTC(),
	}
	fmt.Println("Transaction created:", transaction)
	a.Balance += parsedAmount
	fmt.Println("Balance after deposit:", a.Balance)
	return &transaction, nil
}

// Withdraw removes funds from the account and returns a transaction record.
// The amount is expected to be in dollars, and it will be converted to cents for precision.
// It returns an error if the withdrawal amount is negative or if there are insufficient funds.
func (a *Account) Withdraw(amount float64) (*Transaction, error) {
	fmt.Println("Balance before withdrawal:", a.Balance)
	a.mu.Lock()
	defer a.mu.Unlock()
	// Check if the amount is positive before proceeding with the withdrawal
	if amount < 0 {
		return nil, errors.New("Withdrawal amount must be positive")
	}
	parsedAmount := int64(amount * 100) // Convert to cents for precision
	if parsedAmount > a.Balance {
		return nil, errors.New("Insufficient funds for withdrawal")
	}
	fmt.Println("Withdrawing amount:", amount)
	transaction := Transaction{
		ID:        uuid.New(),
		AccountID: a.ID,
		Amount:    -parsedAmount,
		Created:   time.Now().UTC(),
	}
	fmt.Println("Transaction created:", transaction)
	a.Balance -= parsedAmount
	fmt.Println("Balance after withdrawal:", a.Balance)
	return &transaction, nil
}

// GetBalance returns the current balance of the account in dollars.
// It converts the balance from cents to dollars for display purposes.
func (a *Account) GetBalance() float64 {
	fmt.Println("Getting balance:", a.Balance)
	return float64(a.Balance) / 100 // Convert cents back to dollars
}
