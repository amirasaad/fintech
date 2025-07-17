package handler

import (
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	account "github.com/amirasaad/fintech/pkg/domain/account"
	events "github.com/amirasaad/fintech/pkg/domain/account/events"
	money "github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// NewDepositTransaction creates a deposit transaction record
func NewDepositTransaction(e events.DepositRequestedEvent) *account.Transaction {
	moneyVal, _ := money.New(e.Amount, currency.Code(e.Currency))
	return &account.Transaction{
		ID:          uuid.New(),
		AccountID:   uuid.MustParse(e.AccountID),
		UserID:      uuid.MustParse(e.UserID),
		PaymentID:   e.PaymentID,
		Amount:      moneyVal,
		MoneySource: account.MoneySource(e.Source),
		Status:      account.TransactionStatusInitiated,
		CreatedAt:   time.Now().UTC(),
	}
}

// NewWithdrawTransaction creates a withdrawal transaction record
func NewWithdrawTransaction(e events.WithdrawRequestedEvent, extTarget *account.ExternalTarget) *account.Transaction {
	moneyVal, _ := money.New(e.Amount, currency.Code(e.Currency))
	return &account.Transaction{
		ID:             uuid.New(),
		AccountID:      uuid.MustParse(e.AccountID),
		UserID:         uuid.MustParse(e.UserID),
		PaymentID:      e.PaymentID,
		Amount:         moneyVal.Negate(), // NEGATE for withdraw
		Status:         account.TransactionStatusInitiated,
		ExternalTarget: *extTarget,
		CreatedAt:      time.Now().UTC(),
	}
}

// NewTransferTransactions creates both outgoing and incoming transfer transaction records
func NewTransferTransactions(e events.TransferRequestedEvent, destUserID uuid.UUID) (outTx, inTx *account.Transaction) {
	moneyVal, _ := money.New(e.Amount, currency.Code(e.Currency))
	outTx = &account.Transaction{
		ID:          uuid.New(),
		AccountID:   e.SourceAccountID,
		UserID:      e.SenderUserID,
		Amount:      moneyVal.Negate(),
		MoneySource: account.MoneySource(e.Source),
		Status:      account.TransactionStatusInitiated,
		CreatedAt:   time.Now().UTC(),
	}
	inTx = &account.Transaction{
		ID:          uuid.New(),
		AccountID:   e.DestAccountID,
		UserID:      destUserID, // Set to actual destination user
		Amount:      moneyVal,
		MoneySource: account.MoneySource(e.Source),
		Status:      account.TransactionStatusInitiated,
		CreatedAt:   time.Now().UTC(),
	}
	return
}
