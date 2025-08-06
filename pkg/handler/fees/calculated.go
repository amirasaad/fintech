package fees

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/eventbus"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
	repoaccount "github.com/amirasaad/fintech/pkg/repository/account"
	repotransaction "github.com/amirasaad/fintech/pkg/repository/transaction"
)

// HandleCalculated handles FeesCalculated events.
// It updates the transaction with the calculated fees and deducts them from the account balance.
func HandleCalculated(
	uow repository.UnitOfWork,
	logger *slog.Logger,
) eventbus.HandlerFunc {
	return func(
		ctx context.Context,
		e events.Event,
	) error {
		log := logger.With(
			"handler", "fees.HandleCalculated",
			"event_type", e.Type(),
		)
		log.Info("🟢 [START] Processing FeesCalculated event")

		// Type assert to get the FeesCalculated event
		fc, ok := e.(*events.FeesCalculated)
		if !ok {
			log.Error("unexpected event type")
			return fmt.Errorf("unexpected event type: %s", e.Type())
		}
		log = log.With(
			"transaction_id", fc.TransactionID,
			"event_id", fc.ID,
		)

		return uow.Do(ctx, func(uow repository.UnitOfWork) error {
			// Get transaction repository
			txRepoAny, err := uow.GetRepository((*repotransaction.Repository)(nil))
			if err != nil {
				log.Error("failed to get transaction repository", "error", err)
				return err
			}
			txRepo, ok := txRepoAny.(repotransaction.Repository)
			if !ok {
				log.Error("invalid transaction repository type", "type", txRepoAny)
				return fmt.Errorf("invalid transaction repository type")
			}

			// Get account repository
			accRepoAny, err := uow.GetRepository((*repoaccount.Repository)(nil))
			if err != nil {
				log.Error("failed to get account repository", "error", err)
				return err
			}
			accRepo, ok := accRepoAny.(repoaccount.Repository)
			if !ok {
				log.Error("invalid account repository type", "type", accRepoAny)
				return fmt.Errorf("invalid account repository type")
			}

			// Get the transaction
			tx, err := txRepo.Get(ctx, fc.TransactionID)
			if err != nil {
				log.Error("failed to get transaction", "error", err)
				return err
			}
			i64TotalFee := fc.Fee.Amount.Amount()
			updateTx := dto.TransactionUpdate{Fee: &i64TotalFee}
			if err = txRepo.Update(ctx, tx.ID, updateTx); err != nil {
				log.Error("failed to update transaction with fees", "error", err)
				return err
			}

			// Deduct fees from account balance
			acc, err := accRepo.Get(ctx, tx.AccountID)
			if err != nil {
				log.Error("failed to get account", "error", err)
				return err
			}

			// Convert to domain model to use money operations
			domainAcc := mapper.MapAccountReadToDomain(acc)
			newBalance, err := domainAcc.Balance.Subtract(fc.Fee.Amount)
			if err != nil {
				log.Error(
					"failed to subtract fee",
					"fee", fc.Fee.Amount,
				)
				return err
			}
			i64Balance := newBalance.Amount()
			if err := accRepo.Update(
				ctx,
				acc.ID,
				dto.AccountUpdate{Balance: &i64Balance},
			); err != nil {
				log.Error(
					"failed to update account balance with fees",
					"error", err,
				)
				return err
			}

			log.Info(
				"✅ Fees calculated and deducted",
				"transaction_id", fc.TransactionID,
				"total_fee", fc.Fee.Amount,
				"new_account_balance", i64Balance,
			)
			return nil
		})
	}
}
