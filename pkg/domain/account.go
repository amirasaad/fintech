package domain

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

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

// CurrencyMeta holds metadata for a currency, such as decimals and symbol.
type CurrencyMeta struct {
	Decimals int
	Symbol   string
}

const (
	// DefaultCurrency is the fallback currency code (USD)
	DefaultCurrency = "USD"
	// DefaultDecimals is the default number of decimal places for currencies
	DefaultDecimals = 2
)

// CurrencyInfo maps ISO 4217 currency codes to their metadata.
var CurrencyInfo = map[string]CurrencyMeta{
	"USD": {Decimals: DefaultDecimals, Symbol: "$"},
	"EUR": {Decimals: DefaultDecimals, Symbol: "€"},
	"JPY": {Decimals: 0, Symbol: "¥"},
	"KWD": {Decimals: 3, Symbol: "د.ك"},
	"EGP": {Decimals: DefaultDecimals, Symbol: "£"},
	// Add more as needed
}

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

	// Conversion fields (nullable when no conversion occurs)
	OriginalAmount   *float64 // Amount in original currency
	OriginalCurrency *string  // Original currency code
	ConversionRate   *float64 // Rate used for conversion
}

// iso4217 is a minimal set of supported ISO 4217 currency codes.
var iso4217 = map[string]struct{}{
	"USD": {},
	"EUR": {},
	"GBP": {},
	"JPY": {},
	"KWD": {},
	"EGP": {},
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
		currency = DefaultCurrency
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
		Currency:         currency,
		CreatedAt:        created,
		OriginalAmount:   originalAmount,
		OriginalCurrency: originalCurrency,
		ConversionRate:   conversionRate,
	}
}

// NewTransactionWithCurrency creates a new transaction with the specified currency.
// If the currency is invalid or empty, defaults to USD.
func NewTransactionWithCurrency(id, userID, accountID uuid.UUID, amount, balance int64, currency string) *Transaction {
	if !IsValidCurrencyCode(currency) {
		currency = DefaultCurrency
	}
	return &Transaction{
		ID:               id,
		UserID:           userID,
		AccountID:        accountID,
		Amount:           amount,
		Balance:          balance,
		CreatedAt:        time.Now(),
		Currency:         currency,
		OriginalAmount:   nil,
		OriginalCurrency: nil,
		ConversionRate:   nil,
	}
}

// Deposit adds funds to the account if the currency matches and returns a transaction record.
// Returns an error if the currency does not match or the deposit amount is invalid.
func (a *Account) Deposit(userID uuid.UUID, money Money) (*Transaction, error) {
	if a.UserID != userID {
		return nil, ErrUserUnauthorized
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	if money.Amount <= 0 {
		return nil, ErrTransactionAmountMustBePositive
	}

	amountStr := fmt.Sprintf("%.2f", money.Amount)
	amountRat, ok := new(big.Rat).SetString(amountStr)
	if !ok {
		return nil, fmt.Errorf("invalid amount format")
	}

	meta, ok := CurrencyInfo[money.Currency]
	if !ok {
		meta.Decimals = 2 // default
	}
	multiplier := math.Pow10(meta.Decimals)
	centsRat := new(big.Rat).Mul(amountRat, big.NewRat(int64(multiplier), 1))

	if !centsRat.IsInt() {
		return nil, fmt.Errorf("amount has more than %d decimal places", meta.Decimals)
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
	a.Balance += parsedAmount

	transaction := Transaction{
		ID:        uuid.New(),
		UserID:    userID,
		AccountID: a.ID,
		Amount:    parsedAmount,
		Balance:   a.Balance,
		CreatedAt: time.Now().UTC(),
		Currency:  money.Currency,
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

	if money.Amount <= 0 {
		return nil, ErrWithdrawalAmountMustBePositive
	}

	meta, ok := CurrencyInfo[money.Currency]
	if !ok {
		meta.Decimals = DefaultDecimals
	}
	multiplier := math.Pow10(meta.Decimals)
	cents := int64(math.Round(money.Amount * multiplier))

	if cents > a.Balance {
		return nil, ErrInsufficientFunds
	}
	a.Balance -= cents

	transaction := Transaction{
		ID:        uuid.New(),
		UserID:    userID,
		AccountID: a.ID,
		Amount:    -cents,
		Balance:   a.Balance,
		CreatedAt: time.Now().UTC(),
		Currency:  money.Currency,
	}

	return &transaction, nil
}

// GetBalance returns the current balance of the account in dollars.
// It converts the balance from cents to dollars for display purposes.
func (a *Account) GetBalance(userID uuid.UUID) (balance float64, err error) {
	if a.UserID != userID {
		err = ErrUserUnauthorized
		return
	}
	meta, ok := CurrencyInfo[a.Currency]
	if !ok {
		meta.Decimals = DefaultDecimals
	}
	divisor := math.Pow10(meta.Decimals)
	balance = float64(a.Balance) / divisor
	return
}
