package account

import (
	"time"

	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

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

// TransactionStatus represents the status of a transaction in the payment lifecycle.
type TransactionStatus string

// Transaction status constants define the lifecycle of a transaction.
const (
	// TransactionStatusPending indicates that a transaction
	// has been initiated and is awaiting completion.
	TransactionStatusPending TransactionStatus = "pending"
	// TransactionStatusCompleted indicates that a transaction
	// has been completed successfully.
	TransactionStatusCompleted TransactionStatus = "completed"
	// TransactionStatusFailed indicates that a transaction
	// has been failed.
	TransactionStatusFailed TransactionStatus = "failed"
)

// ExternalTarget represents the destination for an external withdrawal,
// such as a bank account or wallet.
type ExternalTarget struct {
	BankAccountNumber     string
	RoutingNumber         string
	ExternalWalletAddress string
}

// Transaction represents a financial transaction, capturing all details of a
// single ledger entry.
// It acts as a value object within the domain.
type Transaction struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	AccountID      uuid.UUID
	Amount         money.Money
	Balance        money.Money // A snapshot of the account balance at the time of the transaction.
	MoneySource    MoneySource // The origin of the funds (e.g., Cash, BankAccount, Stripe).
	Status         TransactionStatus
	ExternalTarget ExternalTarget
	PaymentID      string // The external payment provider's ID for webhook correlation.
	CreatedAt      time.Time
	// TargetCurrency specifies the currency the account is credited in,
	// which is crucial for multi-currency deposits.
	TargetCurrency string
	// ConversionInfo holds details about a transaction.
	ConversionInfo *common.ConversionInfo
}

// NewTransactionFromData creates a Transaction instance from raw data.
// This function is intended for use by repositories to hydrate a domain object from a data store
// or for setting up test fixtures.
// It bypasses domain invariants and should not be used in business logic.
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
