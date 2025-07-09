package domain

import (
	"errors"
	"math"
	"regexp"
	"sync"
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/google/uuid"
)

var (
	// ErrDepositAmountExceedsMaxSafeInt is returned when a deposit would overflow the account balance.
	ErrDepositAmountExceedsMaxSafeInt = errors.New("deposit amount exceeds maximum safe integer value") // Deposit would overflow balance

	// ErrTransactionAmountMustBePositive is returned when a transaction amount is not positive.
	ErrTransactionAmountMustBePositive = errors.New("transaction amount must be positive") // Amount must be > 0

	// ErrWithdrawalAmountMustBePositive is returned when a withdrawal amount is not positive.
	ErrWithdrawalAmountMustBePositive = errors.New("withdrawal amount must be positive") // Withdrawal must be > 0

	// ErrInsufficientFunds is returned when an account has insufficient funds for a withdrawal.
	ErrInsufficientFunds = errors.New("insufficient funds for withdrawal") // Not enough balance

	// ErrAccountNotFound is returned when an account cannot be found.
	ErrAccountNotFound = errors.New("account not found") // Account does not exist

	// ErrInvalidCurrencyCode is returned when a currency code is invalid.
	ErrInvalidCurrencyCode = errors.New("invalid currency code") // Currency code not recognized
)

// ConversionInfo holds details about a currency conversion performed during a transaction.
type ConversionInfo struct {
	OriginalAmount    float64
	OriginalCurrency  string
	ConvertedAmount   float64
	ConvertedCurrency string
	ConversionRate    float64
}

// Account represents a user's financial account, supporting multi-currency.
type Account struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Balance   int64 // Account balance snapshot
	UpdatedAt time.Time
	CreatedAt time.Time
	Currency  currency.Code // ISO 4217 currency code
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

	// Conversion fields (nullable when no conversion occurs)
	OriginalAmount   *float64 // Amount in original currency
	OriginalCurrency *string  // Original currency code
	ConversionRate   *float64 // Rate used for conversion
}

// IsValidCurrencyFormat returns true if the code is a well-formed ISO 4217 currency code (3 uppercase letters).
func IsValidCurrencyFormat(code string) bool {
	re := regexp.MustCompile(`^[A-Z]{3}$`)
	return re.MatchString(code)
}

// NewTransactionWithCurrency creates a new transaction with the specified currency.
func NewAccount(userID uuid.UUID) (acc *Account) {
	acc, _ = NewAccountWithCurrency(userID, currency.DefaultCurrency)
	return
}

// NewAccountWithCurrency creates a new account with the specified currency.
// If the currency is invalid or empty, defaults to USD.
func NewAccountWithCurrency(userID uuid.UUID, currencyCode currency.Code) (acc *Account, err error) {
	if !currency.IsValidCurrencyFormat(string(currencyCode)) {
		return nil, ErrInvalidCurrencyCode
	}
	return &Account{
		ID:        uuid.New(),
		UserID:    userID,
		CreatedAt: time.Now(),
		Balance:   0,
		Currency:  currencyCode,
		mu:        sync.Mutex{},
	}, nil
}

// NewAccountFromData creates an Account from raw data (used for DB hydration).
func NewAccountFromData(
	id, userID uuid.UUID,
	balance int64,
	currencyCode currency.Code,
	created, updated time.Time,
) *Account {
	return &Account{
		ID:        id,
		UserID:    userID,
		Balance:   balance,
		Currency:  currencyCode,
		CreatedAt: created,
		UpdatedAt: updated,
		mu:        sync.Mutex{},
	}
}

// NewTransactionFromData creates a Transaction from raw data (used for DB hydration or test fixtures).
// This constructor should NOT be used in service, API, or domain logic outside of repository hydration or tests.
// All business transaction creation must go through Account aggregate methods (Deposit, Withdraw, etc.).
func NewTransactionFromData(
	id, userID, accountID uuid.UUID,
	amount, balance int64,
	currencyCode string,
	created time.Time,
	originalAmount *float64,
	originalCurrency *string,
	conversionRate *float64,
) *Transaction {
	return &Transaction{
		ID:               id,
		UserID:           userID,
		AccountID:        accountID,
		Amount:           amount,
		Balance:          balance,
		Currency:         currencyCode,
		CreatedAt:        created,
		OriginalAmount:   originalAmount,
		OriginalCurrency: originalCurrency,
		ConversionRate:   conversionRate,
	}
}

// NewTransactionWithCurrency creates a new transaction with the specified currency.
// This is intended for internal use by the Account aggregate, or for test setup.
// Do NOT use this directly in services, API, or other domain logic—use Account methods instead.
func NewTransactionWithCurrency(id, userID, accountID uuid.UUID, amount, balance int64, currencyCode string) *Transaction {
	if !IsValidCurrencyFormat(currencyCode) {
		currencyCode = currency.DefaultCurrency
	}
	return &Transaction{
		ID:               id,
		UserID:           userID,
		AccountID:        accountID,
		Amount:           amount,
		Balance:          balance,
		CreatedAt:        time.Now(),
		Currency:         currencyCode,
		OriginalAmount:   nil,
		OriginalCurrency: nil,
		ConversionRate:   nil,
	}
}

// GetBalance returns the current balance of the account in dollars.
// It converts the balance from cents to dollars for display purposes.
func (a *Account) GetBalance(userID uuid.UUID) (balance float64, err error) {
	if a.UserID != userID {
		err = ErrUserUnauthorized
		return
	}
	meta, err := currency.Get(string(a.Currency))
	if err != nil {
		return 0, err
	}
	divisor := math.Pow10(meta.Decimals)
	balance = float64(a.Balance) / divisor
	return
}

// GetBalanceAsMoney returns the current balance as a Money value object.
func (a *Account) GetBalanceAsMoney(userID uuid.UUID) (money Money, err error) {
	if a.UserID != userID {
		err = ErrUserUnauthorized
		return
	}
	money, err = NewMoneyFromSmallestUnit(a.Balance, a.Currency)
	return
}

// Deposit adds funds to the account if the currency matches and returns a transaction record.
// Returns an error if the currency does not match or the deposit amount is invalid.
func (a *Account) Deposit(userID uuid.UUID, money Money) (*Transaction, error) {
	if a.UserID != userID {
		return nil, ErrUserUnauthorized
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	if !money.IsPositive() {
		return nil, ErrTransactionAmountMustBePositive
	}

	if string(money.Currency()) != string(a.Currency) {
		return nil, ErrInvalidCurrencyCode
	}

	// Check for overflow before performing the addition
	depositAmount := int64(money.Amount())
	if depositAmount > 0 && a.Balance > 0 && depositAmount > math.MaxInt64-a.Balance {
		return nil, ErrDepositAmountExceedsMaxSafeInt
	}

	// Get current balance as Money
	currentBalance, err := a.GetBalanceAsMoney(userID)
	if err != nil {
		return nil, err
	}

	// Add the deposit amount to current balance
	newBalance, err := currentBalance.Add(money)
	if err != nil {
		return nil, err
	}

	// Update account balance
	a.Balance = int64(newBalance.Amount())

	transaction := Transaction{
		ID:        uuid.New(),
		UserID:    userID,
		AccountID: a.ID,
		Amount:    depositAmount,
		Balance:   a.Balance,
		CreatedAt: time.Now().UTC(),
		Currency:  string(money.Currency()),
	}

	return &transaction, nil
}

// Withdraw removes funds from the account if the currency matches and returns a transaction record.
// Returns an error if the currency does not match or if there are insufficient funds.
func (a *Account) Withdraw(userID uuid.UUID, money Money) (*Transaction, error) {
	if a.UserID != userID {
		return nil, ErrUserUnauthorized
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	if !money.IsPositive() {
		return nil, ErrWithdrawalAmountMustBePositive
	}

	// Get current balance as Money
	currentBalance, err := a.GetBalanceAsMoney(userID)
	if err != nil {
		return nil, err
	}

	// Check if we have sufficient funds
	hasEnough, err := currentBalance.GreaterThan(money)
	if err != nil {
		return nil, err
	}
	if !hasEnough && !currentBalance.Equals(money) {
		return nil, ErrInsufficientFunds
	}

	// Subtract the withdrawal amount from current balance
	newBalance, err := currentBalance.Subtract(money)
	if err != nil {
		return nil, err
	}

	// Update account balance
	a.Balance = int64(newBalance.Amount())

	transaction := Transaction{
		ID:        uuid.New(),
		UserID:    userID,
		AccountID: a.ID,
		Amount:    -int64(money.Amount()),
		Balance:   a.Balance,
		CreatedAt: time.Now().UTC(),
		Currency:  string(money.Currency()),
	}

	return &transaction, nil
}
