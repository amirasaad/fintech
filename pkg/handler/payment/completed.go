package payment

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/handler/common"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
)

// ErrInvalidRepositoryType Define custom error for invalid repository type

// HandleCompleted handles PaymentCompletedEvent,
// updates the transaction status in the DB, and publishes a follow-up event if needed.
func HandleCompleted(
	bus eventbus.Bus,
	uow repository.UnitOfWork,
	logger *slog.Logger,
) func(
	ctx context.Context,
	e events.Event,
) error {
	return func(
		ctx context.Context,
		e events.Event,
	) error {
		log := logger.With(
			"handler", "payment.HandleCompleted",
			"event_type", e.Type(),
		)
		log.Info(
			"🟢 [START] HandleCompleted received event",
			"event_type", e.Type(),
		)
		pc, ok := e.(*events.PaymentCompleted)
		if !ok {
			log.Error(
				"Skipping unexpected event type",
				"event", e,
			)
			return nil
		}
		log = log.With(
			"user_id", pc.UserID,
			"account_id", pc.AccountID,
			"payment_id", pc.PaymentID,
			"transaction_id", pc.TransactionID,
		)
		if err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			accRepo, err := common.GetAccountRepository(uow, log)
			if err != nil {
				return err
			}
			txRepo, err := common.GetTransactionRepository(uow, log)
			if err != nil {
				log.Error("failed to get transaction repository", "error", err)
				return err
			}
			tx, err := txRepo.GetByPaymentID(ctx, pc.PaymentID)
			if err != nil {
				log.Error(
					"failed to get transaction by payment ID",
					"error", err,
				)
				return err
			}
			log = log.With(
				"transaction_id", tx.ID,
				"user_id", tx.UserID,
			)
			acc, err := accRepo.Get(ctx, tx.AccountID)
			if err != nil {
				log.Error(
					"failed to get account",
					"error", err,
				)
				return err
			}
			domainAcc, err := mapper.MapAccountReadToDomain(acc)
			if err != nil {
				log.Error(
					"failed to map account to domain",
					"error", err,
				)
				return err
			}

			// Log provider fee details before calculation
			log.Info("💸 Provider fee details",
				"fee_amount_struct", pc.ProviderFee.Amount,
				"fee_amount_cents", pc.ProviderFee.Amount.Amount(),
			)

			// Calculate the net amount after deducting fees
			netAmount, err := pc.Amount.Subtract(pc.ProviderFee.Amount)
			if err != nil {
				log.Error(
					"failed to calculate net amount after fees",
					"error", err,
				)
				return err
			}

			// Update balance with net amount
			newBalance, err := domainAcc.Balance.Add(netAmount)
			if err != nil {
				log.Error(
					"failed to add net transaction amount to balance",
					"error", err,
				)
				return err
			}

			oldStatus := tx.Status
			status := string(account.TransactionStatusCompleted)
			tx.Status = status

			// Store the gross amount in the transaction
			amount := netAmount.Amount()
			currency := pc.Amount.Currency().String()
			balance := newBalance.Amount()
			fee := pc.ProviderFee.Amount.Amount()
			log.Info("💸 Captured provider fee for transaction", "fee_cents", fee)

			update := dto.TransactionUpdate{
				Status:   &status,
				Amount:   &amount,
				Currency: &currency,
				Balance:  &balance,
				Fee:      &fee, // Store the fee with the transaction
			}

			if err = txRepo.Update(ctx, tx.ID, update); err != nil {
				log.Error(
					"failed to update transaction status",
					"error", err,
				)
				return err
			}

			log.Info(
				"✅ [SUCCESS] transaction status updated",
				"old_status", oldStatus,
				"new_status", tx.Status,
			)

			f64Balance := newBalance.Amount()
			if err := accRepo.Update(
				ctx,
				tx.AccountID,
				dto.AccountUpdate{Balance: &f64Balance},
			); err != nil {
				log.Error(
					"failed to update account balance",
					"error", err,
				)
				return err
			}

			log.Info(
				"✅ [SUCCESS] account balance updated",
				"account_id", acc.ID,
				"new_balance", newBalance,
				"balance", domainAcc.Balance,
			)
			return nil
		}); err != nil {
			log.Error(
				"uow.Do failed",
				"error", err,
			)
			return err
		}
		return nil
	}
}
