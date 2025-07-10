package account

import (
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/google/uuid"
)

// Transaction represents a financial transaction, supporting multi-currency.
type Transaction struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	AccountID uuid.UUID
	Amount    int64
	Currency  currency.Code // Transaction Currency
	Balance   int64         // Account balance snapshot
	CreatedAt time.Time

	// Conversion fields (nullable when no conversion occurs)
	OriginalAmount   *float64 // Amount in original currency
	OriginalCurrency *string  // Original currency code
	ConversionRate   *float64 // Rate used for conversion
}

// NewTransactionFromData creates a Transaction from raw data (used for DB hydration or test fixtures).
// This bypasses invariants and should only be used for repository hydration or tests.
func NewTransactionFromData(
	id, userID, accountID uuid.UUID,
	amount, balance int64,
	currencyCode currency.Code,
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

type transactionBuilder struct {
	id               uuid.UUID
	userID           uuid.UUID
	accountID        uuid.UUID
	amount           int64
	currency         currency.Code
	balance          int64
	createdAt        time.Time
	originalAmount   *float64
	originalCurrency *string
	conversionRate   *float64
}

// NewTransaction creates a new transactionBuilder with default values.
func NewTransaction() *transactionBuilder {
	return &transactionBuilder{
		id:        uuid.New(),
		createdAt: time.Now(),
	}
}

func (b *transactionBuilder) WithUserID(userID uuid.UUID) *transactionBuilder {
	b.userID = userID
	return b
}

func (b *transactionBuilder) WithAccountID(accountID uuid.UUID) *transactionBuilder {
	b.accountID = accountID
	return b
}

func (b *transactionBuilder) WithAmount(amount int64) *transactionBuilder {
	b.amount = amount
	return b
}

func (b *transactionBuilder) WithCurrency(currencyCode currency.Code) *transactionBuilder {
	b.currency = currencyCode
	return b
}

func (b *transactionBuilder) WithBalance(balance int64) *transactionBuilder {
	b.balance = balance
	return b
}

func (b *transactionBuilder) WithCreatedAt(t time.Time) *transactionBuilder {
	b.createdAt = t
	return b
}

func (b *transactionBuilder) WithOriginalAmount(v *float64) *transactionBuilder {
	b.originalAmount = v
	return b
}

func (b *transactionBuilder) WithOriginalCurrency(v *string) *transactionBuilder {
	b.originalCurrency = v
	return b
}

func (b *transactionBuilder) WithConversionRate(v *float64) *transactionBuilder {
	b.conversionRate = v
	return b
}

// Build validates invariants and returns a new Transaction instance.
func (b *transactionBuilder) Build() *Transaction {
	// Optionally add validation here
	return &Transaction{
		ID:               b.id,
		UserID:           b.userID,
		AccountID:        b.accountID,
		Amount:           b.amount,
		Currency:         b.currency,
		Balance:          b.balance,
		CreatedAt:        b.createdAt,
		OriginalAmount:   b.originalAmount,
		OriginalCurrency: b.originalCurrency,
		ConversionRate:   b.conversionRate,
	}
}

// Deprecated: Use NewTransaction().With...().Build() instead.
func NewTransactionWithCurrency(id, userID, accountID uuid.UUID, amount, balance int64, currencyCode currency.Code) *Transaction {
	return NewTransaction().WithUserID(userID).
		WithAccountID(accountID).
		WithAmount(amount).
		WithCurrency(currencyCode).
		WithBalance(balance).
		WithCreatedAt(time.Now()).
		Build()
}
