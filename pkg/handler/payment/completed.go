package payment

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
	repoaccount "github.com/amirasaad/fintech/pkg/repository/account"
)

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
			repo, err := uow.TransactionRepository()
			if err != nil {
				log.Error(
					"❌ [ERROR] failed to get transaction repo",
					"error", err,
				)
				return err
			}
			tx, err := repo.GetByPaymentID(pc.PaymentID)
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
			oldStatus := tx.Status
			tx.Status = account.TransactionStatusCompleted
			if err = repo.Update(tx); err != nil {
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
			repoAny, err := uow.GetRepository((*repoaccount.Repository)(nil))
			if err != nil {
				log.Error(
					"❌ [ERROR] failed to get account repo",
					"error", err,
				)
				return err
			}
			accRepo := repoAny.(repoaccount.Repository)
			acc, err := accRepo.Get(ctx, tx.AccountID)
			if err != nil {
				log.Error(
					"❌ [ERROR] failed to get account",
					"error", err,
				)
				return err
			}
			domainAcc := mapper.MapAccountReadToDomain(acc)
			newBalance, err := domainAcc.Balance.Add(tx.Amount)
			if err != nil {
				log.Error(
					"❌ [ERROR] failed to add transaction amount to balance",
					"error", err,
				)
				return err
			}
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
