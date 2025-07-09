package account

import (
	"errors"
	"math"
	"regexp"
	"sync"
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/domain/user"
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
)

// Use common.ConversionInfo and common.ErrInvalidCurrencyCode
// Account represents a user's financial account, supporting multi-currency.
// Invariants:
//   - Only the account owner can perform actions.
//   - Currency must be valid and match the account’s currency.
//   - Balance cannot overflow int64.
//   - Balance cannot be negative.
//   - All operations are thread-safe.
type Account struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Balance   int64         // Account balance snapshot
	Currency  currency.Code // ISO 4217 currency code
	UpdatedAt time.Time
	CreatedAt time.Time
	mu        sync.Mutex
}

// IsValidCurrencyFormat returns true if the code is a well-formed ISO 4217 currency code (3 uppercase letters).
func IsValidCurrencyFormat(code currency.Code) bool {
	re := regexp.MustCompile(`^[A-Z]{3}$`)
	return re.MatchString(string(code))
}

// accountBuilder is used to build Account instances using a fluent API.
type accountBuilder struct {
	id        uuid.UUID
	userID    uuid.UUID
	balance   int64
	currency  currency.Code
	updatedAt time.Time
	createdAt time.Time
}

// New creates a new accountBuilder with default values.
func New() *accountBuilder {
	return &accountBuilder{
		id:        uuid.New(),
		currency:  currency.DefaultCurrency,
		createdAt: time.Now(),
	}
}

// WithUserID sets the user ID for the account.
func (b *accountBuilder) WithUserID(userID uuid.UUID) *accountBuilder {
	b.userID = userID
	return b
}

// WithCurrency sets the currency for the account.
func (b *accountBuilder) WithCurrency(currencyCode currency.Code) *accountBuilder {
	b.currency = currencyCode
	return b
}

// WithBalance sets the initial balance for the account (for test/data hydration only).
func (b *accountBuilder) WithBalance(balance int64) *accountBuilder {
	b.balance = balance
	return b
}

// WithCreatedAt sets the createdAt timestamp (for test/data hydration only).
func (b *accountBuilder) WithCreatedAt(t time.Time) *accountBuilder {
	b.createdAt = t
	return b
}

// WithUpdatedAt sets the updatedAt timestamp (for test/data hydration only).
func (b *accountBuilder) WithUpdatedAt(t time.Time) *accountBuilder {
	b.updatedAt = t
	return b
}

// Build validates invariants and returns a new Account instance.
func (b *accountBuilder) Build() (*Account, error) {
	if !currency.IsValidCurrencyFormat(string(b.currency)) {
		return nil, common.ErrInvalidCurrencyCode
	}
	if b.userID == uuid.Nil {
		return nil, errors.New("userID is required")
	}
	return &Account{
		ID:        b.id,
		UserID:    b.userID,
		Balance:   b.balance,
		Currency:  b.currency,
		CreatedAt: b.createdAt,
		UpdatedAt: b.updatedAt,
		mu:        sync.Mutex{},
	}, nil
}

// Deprecated: Use New().WithUserID(...).WithCurrency(...).Build() instead.
func NewAccount(userID uuid.UUID) (acc *Account) {
	acc, _ = New().WithUserID(userID).Build()
	return
}

// Deprecated: Use New().WithUserID(...).WithCurrency(...).Build() instead.
func NewAccountWithCurrency(userID uuid.UUID, currencyCode currency.Code) (acc *Account, err error) {
	return New().WithUserID(userID).WithCurrency(currencyCode).Build()
}

// NewAccountFromData creates an Account from raw data (used for DB hydration).
// This bypasses invariants and should only be used for repository hydration or tests.
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

// GetBalance returns the current balance of the account in the main currency unit (e.g., dollars for USD).
// Invariants enforced:
//   - Only the account owner can view the balance.
//   - Currency metadata must be valid.
//
// Returns the balance as float64 or an error if any invariant is violated.
func (a *Account) GetBalance(userID uuid.UUID) (balance float64, err error) {
	if a.UserID != userID {
		err = user.ErrUserUnauthorized
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
// Invariants enforced:
//   - Only the account owner can view the balance.
//   - Currency must be valid.
//
// Returns Money or an error if any invariant is violated.
func (a *Account) GetBalanceAsMoney(userID uuid.UUID) (m money.Money, err error) {
	if a.UserID != userID {
		err = user.ErrUserUnauthorized
		return
	}
	m, err = money.NewMoneyFromSmallestUnit(a.Balance, a.Currency)
	return
}

// Deposit adds funds to the account if all business invariants are satisfied.
// Invariants enforced:
//   - Only the account owner can deposit.
//   - Deposit amount must be positive.
//   - Deposit currency must match account currency.
//   - Deposit must not cause integer overflow.
//
// Returns a Transaction or an error if any invariant is violated.
func (a *Account) Deposit(userID uuid.UUID, m money.Money) (tx *Transaction, err error) {
	if a.UserID != userID {
		err = user.ErrUserUnauthorized
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	if !m.IsPositive() {
		err = ErrTransactionAmountMustBePositive
		return
	}

	if string(m.Currency()) != string(a.Currency) {
		err = common.ErrInvalidCurrencyCode
		return
	}

	// Check for overflow before performing the addition
	depositAmount := int64(m.Amount())
	if depositAmount > 0 && a.Balance > 0 && depositAmount > math.MaxInt64-a.Balance {
		err = ErrDepositAmountExceedsMaxSafeInt
		return
	}

	// Get current balance as Money
	var currentBalance money.Money
	currentBalance, err = a.GetBalanceAsMoney(userID)
	if err != nil {
		return
	}

	// Add the deposit amount to current balance
	var newBalance money.Money
	newBalance, err = currentBalance.Add(m)
	if err != nil {
		return
	}

	// Update account balance
	a.Balance = int64(newBalance.Amount())

	tx = &Transaction{
		ID:        uuid.New(),
		UserID:    userID,
		AccountID: a.ID,
		Amount:    depositAmount,
		Currency:  m.Currency(),
		Balance:   a.Balance,
		CreatedAt: time.Now().UTC(),
	}
	return
}

// Withdraw removes funds from the account if all business invariants are satisfied.
// Invariants enforced:
//   - Only the account owner can withdraw.
//   - Withdrawal amount must be positive.
//   - Withdrawal currency must match account currency.
//   - Cannot withdraw more than the current balance.
//
// Returns a Transaction or an error if any invariant is violated.
func (a *Account) Withdraw(userID uuid.UUID, m money.Money) (tx *Transaction, err error) {
	if a.UserID != userID {
		err = user.ErrUserUnauthorized
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	if !m.IsPositive() {
		err = ErrWithdrawalAmountMustBePositive
		return
	}

	// Get current balance as Money
	var currentBalance money.Money
	currentBalance, err = a.GetBalanceAsMoney(userID)
	if err != nil {
		return
	}

	// Check if we have sufficient funds
	var hasEnough bool
	hasEnough, err = currentBalance.GreaterThan(m)
	if err != nil {
		return
	}
	if !hasEnough && !currentBalance.Equals(m) {
		err = ErrInsufficientFunds
		return
	}

	// Subtract the withdrawal amount from current balance
	var newBalance money.Money
	newBalance, err = currentBalance.Subtract(m)
	if err != nil {
		return
	}

	// Update account balance
	a.Balance = int64(newBalance.Amount())

	tx = &Transaction{
		ID:        uuid.New(),
		UserID:    userID,
		AccountID: a.ID,
		Amount:    -int64(m.Amount()),
		Currency:  m.Currency(),
		Balance:   a.Balance,
		CreatedAt: time.Now().UTC(),
	}
	return
}
