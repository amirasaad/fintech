package domain

import (
	"errors"
	"math"
	"sync"
	"time"

	"log/slog"

	"github.com/google/uuid"
)

var (
	ErrDepositAmountExceedsMaxSafeInt  = errors.New("deposit amount exceeds maximum safe integer value")
	ErrTransactionAmountMustBePositive = errors.New("transaction amount must be positive")
	ErrWithdrawalAmountMustBePositive  = errors.New("withdrawal amount must be positive")
	ErrInsufficientFunds               = errors.New("insufficient funds for withdrawal")
	ErrAccountNotFound                 = errors.New("account not found")
	ErrUserUnauthorized                = errors.New("user unauthorized")
)

type Account struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Balance   int64
	CreatedAt time.Time
	UpdatedAt time.Time
	mu        sync.Mutex
}

type Transaction struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	AccountID uuid.UUID
	Amount    int64
	Balance   int64 // Account balance snapshot
	CreatedAt time.Time
}

func NewAccount(userID uuid.UUID) *Account {
	return &Account{
		ID:        uuid.New(),
		UserID:    userID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Balance:   0,
		mu:        sync.Mutex{},
	}
}

func NewAccountFromData(id, userID uuid.UUID, balance int64, created, updated time.Time) *Account {
	return &Account{
		ID:        id,
		UserID:    userID,
		Balance:   balance,
		CreatedAt: created,
		UpdatedAt: updated,
		mu:        sync.Mutex{},
	}
}

func NewTransactionFromData(id, userID, accountID uuid.UUID, amount, balance int64, created time.Time) *Transaction {
	return &Transaction{
		ID:        id,
		UserID:    userID,
		AccountID: accountID,
		Amount:    amount,
		Balance:   balance,
		CreatedAt: created,
	}
}

// Deposit adds funds to the account and returns a transaction record.
// The amount is expected to be in dollars, and it will be converted to cents for precision.
// It returns an error if the deposit amount is negative.
func (a *Account) Deposit(userID uuid.UUID, amount float64) (*Transaction, error) {
	if a.UserID != userID {
		return nil, ErrUserUnauthorized
	}
	slog.Info("Balance before deposit", slog.Int64("balance", a.Balance))
	a.mu.Lock()
	defer a.mu.Unlock()
	// Check if the amount is positive before proceeding with the deposit
	if amount <= 0 {
		return nil, ErrTransactionAmountMustBePositive
	}

	parsedAmount := int64(amount * 100) // Convert to cents for precision

	// Check for overflow after conversion as well
	if a.Balance > math.MaxInt64-parsedAmount {
		return nil, ErrDepositAmountExceedsMaxSafeInt
	}
	slog.Info("Depositing amount", slog.Int64("amount", parsedAmount))
	a.Balance += parsedAmount
	slog.Info("Balance after deposit", slog.Int64("balance", a.Balance))
	transaction := Transaction{
		ID:        uuid.New(),
		UserID:    userID,
		AccountID: a.ID,
		Amount:    parsedAmount,
		Balance:   a.Balance,
		CreatedAt: time.Now().UTC(),
	}
	slog.Info("Transaction created", slog.Any("transaction", transaction))

	return &transaction, nil
}

// Withdraw removes funds from the account and returns a transaction record.
// The amount is expected to be in dollars, and it will be converted to cents for precision.
// It returns an error if the withdrawal amount is negative or if there are insufficient funds.
func (a *Account) Withdraw(userID uuid.UUID, amount float64) (*Transaction, error) {
	if a.UserID != userID {
		return nil, ErrUserUnauthorized
	}
	slog.Info("Balance before withdrawal", slog.Int64("balance", a.Balance))
	a.mu.Lock()
	defer a.mu.Unlock()
	// Check if the amount is positive before proceeding with the withdrawal
	if amount <= 0 {
		return nil, ErrWithdrawalAmountMustBePositive
	}
	parsedAmount := int64(amount * 100) // Convert to cents for precision
	if parsedAmount > a.Balance {
		return nil, ErrInsufficientFunds
	}
	slog.Info("Withdrawing amount", slog.Int64("amount", parsedAmount))
	a.Balance -= parsedAmount
	slog.Info("Balance after withdrawal", slog.Int64("balance", a.Balance))
	transaction := Transaction{
		ID:        uuid.New(),
		UserID:    userID,
		AccountID: a.ID,
		Amount:    -parsedAmount,
		Balance:   a.Balance,
		CreatedAt: time.Now().UTC(),
	}
	slog.Info("Transaction created:", slog.Any("transaction", transaction))

	return &transaction, nil
}

// GetBalance returns the current balance of the account in dollars.
// It converts the balance from cents to dollars for display purposes.
func (a *Account) GetBalance(userID uuid.UUID) (float64, error) {
	if a.UserID != userID {
		return 0, ErrUserUnauthorized
	}
	slog.Info("Getting balance", slog.Int64("balance", a.Balance))
	return float64(a.Balance) / 100, nil // Convert cents back to dollars
}
