package payment

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
	repoaccount "github.com/amirasaad/fintech/pkg/repository/account"
	repotransaction "github.com/amirasaad/fintech/pkg/repository/transaction"
)

// Define custom error for invalid repository type
var ErrInvalidRepositoryType = errors.New("invalid repository type")

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
			"🟢 [HANDLER] HandleCompleted received event",
			"event_type", e.Type(),
		)
		pc, ok := e.(*events.PaymentCompleted)
		if !ok {
			log.Error(
				"❌ [DISCARD] unexpected event type",
				"event", e,
			)
			return nil
		}
		err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			// Get transaction repository
			txRepoAny, err := uow.GetRepository((*repotransaction.Repository)(nil))
			if err != nil {
				log.Error(
					"❌ [ERROR] failed to get transaction repository",
					"error", err,
				)
				return err
			}
			txRepo, ok := txRepoAny.(repotransaction.Repository)
			if !ok {
				log.Error(
					"❌ [ERROR] invalid transaction repository type",
					"type", txRepoAny,
				)
				return ErrInvalidRepositoryType
			}

			// Get account repository
			accRepoAny, err := uow.GetRepository((*repoaccount.Repository)(nil))
			if err != nil {
				log.Error(
					"❌ [ERROR] failed to get account repository",
					"error", err,
				)
				return err
			}
			accRepo, ok := accRepoAny.(repoaccount.Repository)
			if !ok {
				log.Error(
					"❌ [ERROR] invalid account repository type",
					"type", accRepoAny,
				)
				return ErrInvalidRepositoryType
			}

			// Get transaction by payment ID
			tx, err := txRepo.GetByPaymentID(ctx, pc.PaymentID)
			if err != nil {
				log.Error(
					"❌ [ERROR] failed to get transaction by payment ID",
					"error", err,
					"payment_id", pc.PaymentID,
				)
				return err
			}
			log = log.With(
				"transaction_id", tx.ID,
				"user_id", tx.UserID,
				"payment_id", pc.PaymentID,
			)
			// Update transaction status to completed
			oldStatus := tx.Status
			status := string(account.TransactionStatusCompleted)
			tx.Status = status

			// Update transaction in the database
			update := dto.TransactionUpdate{
				Status: &status,
			}

			if err = txRepo.Update(ctx, tx.ID, update); err != nil {
				log.Error(
					"❌ [ERROR] failed to update transaction status",
					"error", err,
				)
				return err
			}

			log.Info(
				"✅ [SUCCESS] transaction status updated",
				"old_status", oldStatus,
				"new_status", tx.Status,
			)

			// Update account balance after payment completion
			acc, err := accRepo.Get(ctx, tx.AccountID)
			if err != nil {
				log.Error(
					"❌ [ERROR] failed to get account",
					"error", err,
				)
				return err
			}

			// Convert to domain model to use money operations
			domainAcc := mapper.MapAccountReadToDomain(acc)

			// Create money object for transaction amount
			txMoney, err := money.New(tx.Amount, domainAcc.Balance.Currency())
			if err != nil {
				log.Error(
					"❌ [ERROR] failed to create money object for transaction amount",
					"error", err,
				)
				return err
			}

			// Calculate new balance
			newBalance, err := domainAcc.Balance.Add(txMoney)
			if err != nil {
				log.Error(
					"❌ [ERROR] failed to add transaction amount to balance",
					"error", err,
				)
				return err
			}

			// Update account balance
			f64Balance := newBalance.AmountFloat()
			if err := accRepo.Update(
				ctx,
				tx.AccountID,
				dto.AccountUpdate{Balance: &f64Balance},
			); err != nil {
				log.Error(
					"❌ [ERROR] failed to update account balance",
					"error", err,
				)
				return err
			}

			log.Info(
				"✅ [SUCCESS] account balance updated",
				"account_id", acc.ID,
				"new_balance", f64Balance,
			)
			return nil
		})
		if err != nil {
			log.Error(
				"❌ [ERROR] transaction failed",
				"error", err,
			)
			return err
		}
		// Optionally: publish a UI/account balance update event
		// return bus.Emit(
		// ctx,
		// events.AccountBalanceUpdatedEvent{
		// UserID: tx.UserID, AccountID: tx.AccountID, NewBalance: ...})
		return nil
	}
}
