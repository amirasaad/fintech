package account

import (
	"time"

	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// TransactionStatus represents the status of a transaction in the payment lifecycle.
type TransactionStatus string

// TransactionStatusInitiated indicates that a transaction has been initiated but not yet completed.
const TransactionStatusInitiated = "initiated"

// TransactionStatusPending indicates that a transaction is pending and awaiting completion.
const TransactionStatusPending = "pending"

// TransactionStatusCompleted indicates that a transaction has been completed successfully.
const TransactionStatusCompleted = "completed"

// TransactionStatusFailed indicates that a transaction has been failed.
const TransactionStatusFailed TransactionStatus = "failed"

// ExternalTarget represents the destination for an external withdrawal, such as a bank account or wallet.
type ExternalTarget struct {
	BankAccountNumber     string
	RoutingNumber         string
	ExternalWalletAddress string
}

// Transaction represents a financial transaction for an account.
type Transaction struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	AccountID      uuid.UUID
	Amount         money.Money
	Balance        money.Money // Account balance snapshot
	MoneySource    MoneySource // Origin of funds (e.g., Cash, BankAccount, Stripe, etc.)
	Status         TransactionStatus
	ExternalTarget ExternalTarget
	PaymentID      string // External payment provider ID for webhook correlation
	CreatedAt      time.Time
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
