package account

import (
	"time"

	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// Transaction represents a financial transaction, supporting multi-currency and tracking money source.
type Transaction struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	AccountID   uuid.UUID
	Amount      money.Money
	Balance     money.Money // Account balance snapshot
	MoneySource MoneySource // Origin of funds (e.g., Cash, BankAccount, Stripe, etc.)
	CreatedAt   time.Time
}

// NewTransactionFromData creates a Transaction from raw data (used for DB hydration or test fixtures).
// This bypasses invariants and should only be used for repository hydration or tests.
func NewTransactionFromData(
	id, userID, accountID uuid.UUID,
	amount money.Money,
	balance money.Money,
	moneySource MoneySource,
	created time.Time,
) *Transaction {
	return &Transaction{
		ID:          id,
		UserID:      userID,
		AccountID:   accountID,
		Amount:      amount,
		Balance:     balance,
		MoneySource: moneySource,
		CreatedAt:   created,
	}
}

// MoneySource represents the origin of funds for a transaction.
type MoneySource string

// Money source constants define the origin of funds for transactions.
const (
	MoneySourceInternal       MoneySource = "Internal"
	MoneySourceBankAccount    MoneySource = "BankAccount"
	MoneySourceCard           MoneySource = "Card"
	MoneySourceCash           MoneySource = "Cash"
	MoneySourceExternalWallet MoneySource = "ExternalWallet"
)
