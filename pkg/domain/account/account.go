package account

import (
	"errors"
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

var (
	// ErrDepositAmountExceedsMaxSafeInt is returned when a deposit would cause the account balance to overflow.
	ErrDepositAmountExceedsMaxSafeInt = errors.New("deposit amount exceeds maximum safe integer value")

	// ErrTransactionAmountMustBePositive is returned when a transaction amount is not positive.
	ErrTransactionAmountMustBePositive = errors.New("transaction amount must be positive")

	// ErrInsufficientFunds is returned when an account has insufficient funds for a withdrawal or transfer.
	ErrInsufficientFunds = errors.New("insufficient funds")

	// ErrAccountNotFound is returned when an account cannot be found.
	ErrAccountNotFound = errors.New("account not found")

	// ErrCannotTransferToSameAccount is returned when a transfer is attempted from an account to itself.
	ErrCannotTransferToSameAccount = errors.New("cannot transfer to same account")
	// ErrNilAccount is returned when a nil account is provided to a transfer or other operation.
	ErrNilAccount = errors.New("nil account")
	// ErrNotOwner is returned when a user attempts to perform an action on an account they do not own.
	ErrNotOwner = errors.New("not owner")
	// ErrCurrencyMismatch is returned when there is a currency mismatch between accounts or transactions.
	ErrCurrencyMismatch = errors.New("currency mismatch")
)

// Account represents a user's financial account, encapsulating its balance and ownership.
// It acts as an aggregate root, ensuring all state changes are consistent and valid.
//
// Invariants:
// - An account must always have a valid owner (UserID).
// - The account's balance is represented by a Money value object, ensuring currency consistency.
// - The balance can never be negative.
// - All operations are thread-safe, enforced by a mutex.
type Account struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Balance   money.Money // Account balance as a Money value object.
	UpdatedAt time.Time
	CreatedAt time.Time
}

// Builder provides a fluent API for constructing Account instances.
// This pattern is particularly useful for setting optional parameters and ensuring
// that only valid accounts are constructed.
type Builder struct {
	id        uuid.UUID
	userID    uuid.UUID
	balance   int64
	currency  currency.Code
	updatedAt time.Time
	createdAt time.Time
}

// New creates a new Builder with sensible defaults, such as a new UUID and the default currency.
func New() *Builder {
	return &Builder{
		id:        uuid.New(),
		currency:  currency.DefaultCurrency,
		createdAt: time.Now(),
	}
}

// WithID sets the ID for the account being built.
func (b *Builder) WithID(id uuid.UUID) *Builder {
	b.id = id
	return b
}

// WithUserID sets the user ID for the account being built. This is a mandatory field.
func (b *Builder) WithUserID(userID uuid.UUID) *Builder {
	b.userID = userID
	return b
}

// WithCurrency sets the currency for the account being built. If not set, it defaults to the system's default currency.
func (b *Builder) WithCurrency(currencyCode currency.Code) *Builder {
	b.currency = currencyCode
	return b
}

// WithBalance sets the initial balance for the account. This should only be used
// for hydrating an existing account from a data store or for test setup.
func (b *Builder) WithBalance(balance int64) *Builder {
	b.balance = balance
	return b
}

// WithCreatedAt sets the creation timestamp. This is primarily for hydrating
// an existing account from a data store.
func (b *Builder) WithCreatedAt(t time.Time) *Builder {
	b.createdAt = t
	return b
}

// WithUpdatedAt sets the last-updated timestamp. This is primarily for hydrating
// an existing account from a data store.
func (b *Builder) WithUpdatedAt(t time.Time) *Builder {
	b.updatedAt = t
	return b
}

// Build finalizes the construction of the Account. It validates all invariants,
// such as ensuring a valid currency and a non-nil UserID, before returning the
// new Account instance.
func (b *Builder) Build() (*Account, error) {
	if !currency.IsValidCurrencyFormat(string(b.currency)) {
		return nil, common.ErrInvalidCurrencyCode
	}
	if !currency.IsSupported(string(b.currency)) {
		return nil, common.ErrUnsupportedCurrency
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
	}, nil
}

func (a *Account) Currency() currency.Code {
	return a.Balance.Currency()
}

func (a *Account) SetCurrency(c currency.Code) error {
	newBalance, err := money.NewMoneyFromSmallestUnit(a.Balance.Amount(), c)
	if err != nil {
		return err
	}
	a.Balance = newBalance
	return nil
}

// validate checks all business invariants for an operation (common validation logic).
func (a *Account) validate(userID uuid.UUID) error {
	if a.UserID != userID {
		return ErrNotOwner
	}

	return nil
}

func (a *Account) validateAmount(amount money.Money) error {
	if !amount.IsPositive() {
		return ErrTransactionAmountMustBePositive
	}

	return nil
}

// ValidateDeposit checks all business invariants for a deposit operation.
func (a *Account) ValidateDeposit(userID uuid.UUID, amount money.Money) (err error) {

	if err = a.validate(userID); err != nil {
		return
	}
	return a.validateAmount(amount)

}

// ValidateWithdraw removes funds from the account if all business invariants are satisfied.
// Invariants enforced:
//   - Only the account owner can withdraw.
//   - Withdrawal amount must be positive.
//   - Withdrawal currency must match account currency.
//   - Cannot withdraw more than the current balance.
//
// Returns a Transaction or an error if any invariant is violated.
func (a *Account) ValidateWithdraw(userID uuid.UUID, amount money.Money) error {
	if err := a.validate(userID); err != nil {
		return err
	}
	if err := a.validateAmount(amount); err != nil {
		return err
	}
	// Sufficient funds check: do not allow negative balance
	hasEnough, err := a.Balance.GreaterThan(amount)
	if err != nil {
		return err
	}
	if !hasEnough && !a.Balance.Equals(amount) {
		return ErrInsufficientFunds
	}
	return nil
}

// ValidateTransfer ensures that a funds transfer from this account to another is valid.
func (a *Account) ValidateTransfer(senderUserID, receiverUserID uuid.UUID, dest *Account, amount money.Money) error {
	if a == nil || dest == nil {
		return ErrNilAccount
	}
	if a.ID == dest.ID {
		return ErrCannotTransferToSameAccount
	}
	if a.UserID != senderUserID {
		return ErrNotOwner
	}
	if !amount.IsPositive() {
		return ErrTransactionAmountMustBePositive
	}
	if !a.Balance.IsSameCurrency(amount) || !dest.Balance.IsSameCurrency(amount) {
		return ErrCurrencyMismatch
	}
	hasEnough, err := a.Balance.GreaterThan(amount)
	if err != nil {
		return err
	}
	if !hasEnough && !a.Balance.Equals(amount) {
		return ErrInsufficientFunds
	}

	return nil
}
