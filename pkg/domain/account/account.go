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

	// ErrCannotTransferToSameAccount is returned when a transfer is attempted to the same account.
	ErrCannotTransferToSameAccount = errors.New("cannot transfer to same account")
	// ErrNilAccount is returned when a nil account is encountered in a transfer or operation.
	ErrNilAccount = errors.New("nil account")
	// ErrNotOwner is returned when an operation is attempted by a non-owner.
	ErrNotOwner = errors.New("not owner")
	// ErrCurrencyMismatch is returned when there is a currency mismatch in an operation.
	ErrCurrencyMismatch = errors.New("currency mismatch")
)

// Account represents a user's financial account, supporting multi-currency.
// Invariants:
//   - Only the account owner can perform actions.
//   - Currency must be valid and match the account's currency.
//   - Balance cannot overflow int64.
//   - Balance cannot be negative.
//   - All operations are thread-safe.
type Account struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Balance   money.Money // Account balance as a value object
	UpdatedAt time.Time
	CreatedAt time.Time
	mu        sync.Mutex
}

// IsValidCurrencyFormat returns true if the code is a well-formed ISO 4217 currency code (3 uppercase letters).
func IsValidCurrencyFormat(code currency.Code) bool {
	re := regexp.MustCompile(`^[A-Z]{3}$`)
	return re.MatchString(string(code))
}

// Builder is used to build Account instances using a fluent API.
type Builder struct {
	id        uuid.UUID
	userID    uuid.UUID
	balance   int64
	currency  currency.Code
	updatedAt time.Time
	createdAt time.Time
}

// New creates a new Builder with default values.
func New() *Builder {
	return &Builder{
		id:        uuid.New(),
		currency:  currency.DefaultCurrency,
		createdAt: time.Now(),
	}
}

// WithUserID sets the user ID for the account.
func (b *Builder) WithUserID(userID uuid.UUID) *Builder {
	b.userID = userID
	return b
}

// WithCurrency sets the currency for the account.
func (b *Builder) WithCurrency(currencyCode currency.Code) *Builder {
	b.currency = currencyCode
	return b
}

// WithBalance sets the initial balance for the account (for test/data hydration only).
func (b *Builder) WithBalance(balance int64) *Builder {
	b.balance = balance
	return b
}

// WithCreatedAt sets the createdAt timestamp (for test/data hydration only).
func (b *Builder) WithCreatedAt(t time.Time) *Builder {
	b.createdAt = t
	return b
}

// WithUpdatedAt sets the updatedAt timestamp (for test/data hydration only).
func (b *Builder) WithUpdatedAt(t time.Time) *Builder {
	b.updatedAt = t
	return b
}

// Build validates invariants and returns a new Account instance.
func (b *Builder) Build() (*Account, error) {
	if !currency.IsValidCurrencyFormat(string(b.currency)) {
		return nil, common.ErrInvalidCurrencyCode
	}
	if b.userID == uuid.Nil {
		return nil, errors.New("userID is required")
	}
	bal, err := money.NewMoneyFromSmallestUnit(b.balance, b.currency)
	if err != nil {
		return nil, err
	}
	return &Account{
		ID:        b.id,
		UserID:    b.userID,
		Balance:   bal,
		CreatedAt: b.createdAt,
		UpdatedAt: b.updatedAt,
		mu:        sync.Mutex{},
	}, nil
}

// NewAccount creates a new Account for the given user ID.
// Deprecated: Use New().WithUserID(...).WithCurrency(...).Build() instead.
func NewAccount(userID uuid.UUID) (acc *Account) {
	acc, _ = New().WithUserID(userID).Build()
	return
}

// NewAccountWithCurrency creates a new Account for the given user ID and currency.
// Deprecated: Use New().WithUserID(...).WithCurrency(...).Build() instead.
func NewAccountWithCurrency(userID uuid.UUID, currencyCode currency.Code) (acc *Account, err error) {
	return New().WithUserID(userID).WithCurrency(currencyCode).Build()
}

// NewAccountFromData creates an Account from raw data (used for DB hydration).
// This bypasses invariants and should only be used for repository hydration or tests.
func NewAccountFromData(
	id, userID uuid.UUID,
	balance money.Money,
	created, updated time.Time,
) *Account {
	return &Account{
		ID:        id,
		UserID:    userID,
		Balance:   balance,
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
	meta, err := currency.Get(string(a.Balance.Currency()))
	if err != nil {
		return 0, err
	}
	divisor := math.Pow10(meta.Decimals)
	balance = float64(a.Balance.Amount()) / divisor
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
	m = a.Balance
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
func (a *Account) Deposit(userID uuid.UUID, m money.Money, moneySource MoneySource) (tx *Transaction, err error) {
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

	if string(m.Currency()) != string(a.Balance.Currency()) {
		err = common.ErrInvalidCurrencyCode
		return
	}

	// Check for overflow before performing the addition
	depositAmount := int64(m.Amount())
	if depositAmount > 0 && a.Balance.Amount() > 0 && depositAmount > math.MaxInt64-a.Balance.Amount() {
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
	a.Balance = newBalance

	tx = &Transaction{
		ID:          uuid.New(),
		UserID:      userID,
		AccountID:   a.ID,
		Amount:      m,
		Balance:     a.Balance,
		MoneySource: moneySource,
		CreatedAt:   time.Now().UTC(),
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
func (a *Account) Withdraw(userID uuid.UUID, m money.Money, moneySource MoneySource) (tx *Transaction, err error) {
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
	a.Balance = newBalance

	tx = &Transaction{
		ID:          uuid.New(),
		UserID:      userID,
		AccountID:   a.ID,
		Amount:      m.Negate(),
		Balance:     a.Balance,
		MoneySource: moneySource,
		CreatedAt:   time.Now().UTC(),
	}
	return
}

// Transfer moves funds from this account to another account.
func (a *Account) Transfer(initiatorUserID uuid.UUID, dest *Account, amount money.Money, moneySource MoneySource) (txIn, txOut *Transaction, err error) {
	if a == nil || dest == nil {
		return nil, nil, ErrNilAccount
	}
	if a.ID == dest.ID {
		return nil, nil, ErrCannotTransferToSameAccount
	}
	if a.UserID != initiatorUserID {
		return nil, nil, ErrNotOwner
	}
	if !amount.IsPositive() {
		return nil, nil, ErrTransactionAmountMustBePositive
	}
	if !a.Balance.IsSameCurrency(amount) || !dest.Balance.IsSameCurrency(amount) {
		return nil, nil, ErrCurrencyMismatch
	}
	hasEnough, _ := a.Balance.GreaterThan(amount)
	if !hasEnough && !a.Balance.Equals(amount) {
		return nil, nil, ErrInsufficientFunds
	}
	nb, err := a.Balance.Subtract(amount)
	if err != nil {
		return nil, nil, err
	}
	db, err := dest.Balance.Add(amount)
	if err != nil {
		return nil, nil, err
	}
	a.Balance = nb
	dest.Balance = db
	txOut = &Transaction{
		ID:          uuid.New(),
		UserID:      a.UserID,
		AccountID:   a.ID,
		Amount:      amount.Negate(),
		Balance:     a.Balance,
		MoneySource: moneySource, // Set to 'Internal' by caller
		CreatedAt:   time.Now().UTC(),
	}
	txIn = &Transaction{
		ID:          uuid.New(),
		UserID:      dest.UserID,
		AccountID:   dest.ID,
		Amount:      amount,
		Balance:     dest.Balance,
		MoneySource: moneySource, // Set to 'Internal' by caller
		CreatedAt:   time.Now().UTC(),
	}
	return txIn, txOut, nil
}
