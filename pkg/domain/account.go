package domain

import (
	"errors"
	"fmt"
	"math"
	"math/big"
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
	ErrInvalidCurrencyCode             = errors.New("invalid currency code")
	ErrCurrencyMismatch                = errors.New("currency mismatch")
)

// Account represents a user's financial account, supporting multi-currency.
type Account struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Balance   int64 // Account balance snapshot
	UpdatedAt time.Time
	CreatedAt time.Time
	Currency  string // ISO 4217 currency code
	mu        sync.Mutex
}

// Transaction represents a financial transaction, supporting multi-currency.
type Transaction struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	AccountID uuid.UUID
	Amount    int64
	Balance   int64 // Account balance snapshot
	CreatedAt time.Time
	Currency  string // ISO 4217 currency code
}

// iso4217 is a minimal set of supported ISO 4217 currency codes.
var iso4217 = map[string]struct{}{
	"USD": {},
	"EUR": {},
	"GBP": {},
	// Add more as needed
}

// IsValidCurrencyCode returns true if the code is a supported ISO 4217 currency code.
func IsValidCurrencyCode(code string) bool {
	_, ok := iso4217[code]
	return ok
}

// NewAccount creates a new account with default currency USD.
func NewAccount(userID uuid.UUID) *Account {
	return NewAccountWithCurrency(userID, "USD")
}

// NewAccountWithCurrency creates a new account with the specified currency.
// If the currency is invalid or empty, defaults to USD.
func NewAccountWithCurrency(userID uuid.UUID, currency string) *Account {
	if !IsValidCurrencyCode(currency) {
		currency = "USD"
	}
	return &Account{
		ID:        uuid.New(),
		UserID:    userID,
		CreatedAt: time.Now(),
		Balance:   0,
		Currency:  currency,
		mu:        sync.Mutex{},
	}
}

func NewAccountFromData(
	id, userID uuid.UUID,
	balance int64,
	currency string,
	created, updated time.Time,
) *Account {
	return &Account{
		ID:        id,
		UserID:    userID,
		Balance:   balance,
		Currency:  currency,
		CreatedAt: created,
		UpdatedAt: updated,
		mu:        sync.Mutex{},
	}
}

func NewTransactionFromData(
	id, userID, accountID uuid.UUID,
	amount, balance int64,
	currency string,
	created time.Time,
) *Transaction {
	return &Transaction{
		ID:        id,
		UserID:    userID,
		AccountID: accountID,
		Amount:    amount,
		Balance:   balance,
		Currency:  currency,
		CreatedAt: created,
	}
}

// NewTransactionWithCurrency creates a new transaction with the specified currency.
// If the currency is invalid or empty, defaults to USD.
func NewTransactionWithCurrency(id, userID, accountID uuid.UUID, amount, balance int64, currency string) *Transaction {
	if !IsValidCurrencyCode(currency) {
		currency = "USD"
	}
	return &Transaction{
		ID:        id,
		UserID:    userID,
		AccountID: accountID,
		Amount:    amount,
		Balance:   balance,
		CreatedAt: time.Now(),
		Currency:  currency,
	}
}

// Deposit adds funds to the account and returns a transaction record.
// The amount is expected to be in dollars, and it will be converted to cents for precision.
// It returns an error if the deposit amount is negative.
func (a *Account) Deposit(
	userID uuid.UUID,
	amount float64,
) (*Transaction, error) {
	if a.UserID != userID {
		return nil, ErrUserUnauthorized
	}
	slog.Info("Balance before deposit", slog.Int64("balance", a.Balance))
	a.mu.Lock()
	defer a.mu.Unlock()

	if amount <= 0 {
		return nil, ErrTransactionAmountMustBePositive
	}

	amountStr := fmt.Sprintf("%.2f", amount)
	amountRat, ok := new(big.Rat).SetString(amountStr)
	if !ok {
		return nil, fmt.Errorf("invalid amount format")
	}

	multiplier := big.NewRat(100, 1)
	centsRat := new(big.Rat).Mul(amountRat, multiplier)

	if !centsRat.IsInt() {
		return nil, fmt.Errorf("amount has more than 2 decimal places")
	}

	cents := centsRat.Num()

	if cents.Sign() <= 0 {
		return nil, ErrTransactionAmountMustBePositive
	}

	max := big.NewInt(math.MaxInt64)
	balanceBig := big.NewInt(a.Balance)
	newBalance := new(big.Int).Add(balanceBig, cents)

	if newBalance.Cmp(max) > 0 {
		return nil, ErrDepositAmountExceedsMaxSafeInt
	}

	parsedAmount := cents.Int64()
	slog.Info("Depositing amount", slog.Int64("amount", parsedAmount))
	a.Balance += parsedAmount
	slog.Info("Balance after deposit", slog.Int64("balance", a.Balance))

	transaction := Transaction{
		ID:        uuid.New(),
		UserID:    userID,
		AccountID: a.ID,
		Amount:    parsedAmount,
		Balance:   a.Balance,
		Currency:  a.Currency,
		CreatedAt: time.Now().UTC(),
	}
	slog.Info("Transaction created", slog.Any("transaction", transaction))

	return &transaction, nil
}

// Withdraw removes funds from the account and returns a transaction record.
// The amount is expected to be in dollars, and it will be converted to cents for precision.
// It returns an error if the withdrawal amount is negative or if there are insufficient funds.
func (a *Account) Withdraw(
	userID uuid.UUID,
	amount float64,
) (*Transaction, error) {
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
	cents := int64(math.Round(amount * 100))
	if cents > a.Balance {
		return nil, ErrInsufficientFunds
	}
	slog.Info("Withdrawing amount", slog.Int64("amount", cents))
	a.Balance -= cents
	slog.Info("Balance after withdrawal", slog.Int64("balance", a.Balance))
	transaction := Transaction{
		ID:        uuid.New(),
		UserID:    userID,
		AccountID: a.ID,
		Amount:    -cents,
		Balance:   a.Balance,
		Currency:  a.Currency,
		CreatedAt: time.Now().UTC(),
	}
	slog.Info("Transaction created:", slog.Any("transaction", transaction))

	return &transaction, nil
}

// GetBalance returns the current balance of the account in dollars.
// It converts the balance from cents to dollars for display purposes.
func (a *Account) GetBalance(userID uuid.UUID) (balance float64, err error) {
	if a.UserID != userID {
		err = ErrUserUnauthorized
		return
	}
	slog.Info("Getting balance", slog.Int64("balance", a.Balance))
	balance = float64(a.Balance) / 100
	return
}

// DepositWithCurrency adds funds to the account if the currency matches.
// Returns an error if the currency does not match.
func (a *Account) DepositWithCurrency(
	userID uuid.UUID,
	amount float64,
	currency string,
) (
	tx *Transaction,
	err error,
) {
	if !IsValidCurrencyCode(currency) {
		err = ErrInvalidCurrencyCode
		return
	}
	if a.Currency != currency {
		err = fmt.Errorf("%w: account has %s, operation is %s", ErrCurrencyMismatch, a.Currency, currency)
		return
	}
	tx, err = a.Deposit(userID, amount)
	return
}

func (a *Account) WithdrawWithCurrency(
	userID uuid.UUID,
	amount float64,
	currency string,
) (tx *Transaction,
	err error,
) {
	if !IsValidCurrencyCode(currency) {
		err = ErrInvalidCurrencyCode
		return
	}
	if a.Currency != currency {
		err = fmt.Errorf("%w: account has %s, operation is %s", ErrCurrencyMismatch, a.Currency, currency)
		return
	}
	tx, err = a.Withdraw(userID, amount)

	return
}
