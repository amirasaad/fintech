package fees

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/money"
	repoaccount "github.com/amirasaad/fintech/pkg/repository/account"
	repotransaction "github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
)

// FeeCalculator handles fee calculation and application to transactions and accounts
type FeeCalculator struct {
	txRepo  repotransaction.Repository
	accRepo repoaccount.Repository
	logger  *slog.Logger
}

// NewFeeCalculator creates a new FeeCalculator instance
// Returns nil if any of the required parameters are nil
func NewFeeCalculator(
	txRepo repotransaction.Repository,
	accRepo repoaccount.Repository,
	logger *slog.Logger,
) *FeeCalculator {
	if txRepo == nil || accRepo == nil {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &FeeCalculator{
		txRepo:  txRepo,
		accRepo: accRepo,
		logger:  logger,
	}
}

// ApplyFees applies the calculated fees to a transaction and updates the account balance
func (fc *FeeCalculator) ApplyFees(
	ctx context.Context,
	transactionID uuid.UUID,
	fee account.Fee,
) error {
	// Get the transaction
	tx, err := fc.txRepo.Get(ctx, transactionID)
	if err != nil {
		fc.logger.Error("failed to get transaction", "error", err, "transaction_id", transactionID)
		return err
	}

	// Update transaction with new fee
	if err := fc.updateTransactionFee(ctx, tx, fee); err != nil {
		return err
	}

	// Update account balance with fee deduction
	if err := fc.updateAccountBalance(ctx, tx.AccountID, fee.Amount); err != nil {
		return err
	}

	return nil
}

// updateTransactionFee updates a transaction with the calculated fee
func (fc *FeeCalculator) updateTransactionFee(
	ctx context.Context,
	tx *dto.TransactionRead,
	fee account.Fee,
) error {
	// Validate currency is set
	if tx.Currency == "" {
		err := fmt.Errorf("transaction %s has no currency set", tx.ID)
		fc.logger.Error("transaction has no currency",
			"error", err,
			"transaction_id", tx.ID,
		)
		return err
	}

	// Convert existing fee to money type
	txFee, err := money.New(tx.Fee, money.Code(tx.Currency))
	if err != nil {
		fc.logger.Error("invalid transaction fee amount",
			"error", err,
			"transaction_id", tx.ID,
			"fee", tx.Fee,
			"currency", tx.Currency,
		)
		return fmt.Errorf("invalid transaction fee amount: %w", err)
	}

	// Add the new fee
	totalFee, err := txFee.Add(fee.Amount)
	if err != nil {
		fc.logger.Error("failed to add fees",
			"error", err,
			"transaction_id", tx.ID,
			"existing_fee", txFee,
			"new_fee", fee.Amount,
		)
		return fmt.Errorf("failed to add fees: %w", err)
	}

	// Update the transaction
	totalFeeAmount := totalFee.Amount()
	updateTx := dto.TransactionUpdate{Fee: &totalFeeAmount}

	if err := fc.txRepo.Update(ctx, tx.ID, updateTx); err != nil {
		fc.logger.Error("failed to update transaction with fees",
			"error", err,
			"transaction_id", tx.ID,
			"fee", fee.Amount,
		)
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	fc.logger.Info("updated transaction with fee",
		"transaction_id", tx.ID,
		"total_fee", totalFee,
	)
	return nil
}

// updateAccountBalance updates an account balance by deducting the fee
func (fc *FeeCalculator) updateAccountBalance(
	ctx context.Context,
	accountID uuid.UUID,
	feeAmount *money.Money,
) error {
	// Get the account
	acc, err := fc.accRepo.Get(ctx, accountID)
	if err != nil {
		fc.logger.Error("failed to get account", "error", err, "account_id", accountID)
		return err
	}

	// Convert to domain model to use money operations
	domainAcc, err := mapper.MapAccountReadToDomain(acc)
	if err != nil {
		fc.logger.Error("error creating account from dto", "error", err, "account_id", accountID)
		return err
	}

	// Calculate new balance
	newBalance, err := domainAcc.Balance.Subtract(feeAmount)
	if err != nil {
		fc.logger.Error("failed to subtract fee",
			"fee", feeAmount,
			"current_balance", domainAcc.Balance,
			"account_id", accountID,
		)
		return fmt.Errorf("failed to subtract fee from balance: %w", err)
	}

	// Update account balance
	balanceAmount := newBalance.Amount()
	if err := fc.accRepo.Update(
		ctx,
		acc.ID,
		dto.AccountUpdate{Balance: &balanceAmount},
	); err != nil {
		fc.logger.Error("failed to update account balance",
			"error", err,
			"account_id", accountID,
			"new_balance", balanceAmount,
		)
		return err
	}

	fc.logger.Info("updated account balance with fee deduction",
		"account_id", accountID,
		"fee_deducted", feeAmount,
		"new_balance", balanceAmount,
	)
	return nil
}
