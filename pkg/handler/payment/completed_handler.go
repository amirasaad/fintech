package payment

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/domain"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	dto "github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
	repoaccount "github.com/amirasaad/fintech/pkg/repository/account"
)

// CompletedHandler handles PaymentCompletedEvent, updates the transaction status in the DB, and publishes a follow-up event if needed.
func CompletedHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		logger.Info("CompletedHandler: received event", "event", e)
		pe, ok := e.(*events.PaymentCompletedEvent)
		if !ok {
			logger.Error("CompletedHandler: unexpected event type", "event", e)
			return
		}
		err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			repo, err := uow.TransactionRepository()
			if err != nil {
				logger.Error("CompletedHandler: failed to get transaction repo", "error", err)
				return err
			}
			tx, err := repo.GetByPaymentID(pe.PaymentID)
			if err != nil {
				logger.Error("CompletedHandler: failed to get transaction by payment ID", "error", err, "payment_id", pe.PaymentID)
				return err
			}
			logger = logger.With("transaction_id", tx.ID, "user_id", tx.UserID, "payment_id", pe.PaymentID)
			oldStatus := tx.Status
			tx.Status = accountdomain.TransactionStatus(pe.Status)
			if err := repo.Update(tx); err != nil {
				logger.Error("CompletedHandler: failed to update transaction status", "error", err)
				return err
			}
			logger.Info("CompletedHandler: transaction status updated", "old_status", oldStatus, "new_status", tx.Status)
			// Update account balance after payment completion
			repoAny, err := uow.GetRepository((*repoaccount.Repository)(nil))
			if err != nil {
				logger.Error("CompletedHandler: failed to get account repo", "error", err)
				return err
			}
			accRepo := repoAny.(repoaccount.Repository)
			acc, err := accRepo.Get(ctx, tx.AccountID)
			if err != nil {
				logger.Error("CompletedHandler: failed to get account", "error", err)
				return err
			}
			domainAcc := mapper.MapAccountReadToDomain(acc)
			newBalance, err := domainAcc.Balance.Add(tx.Amount)
			if err != nil {
				logger.Error("CompletedHandler: failed to add transaction amount to balance", "error", err)
				return err
			}
			f64Balance := newBalance.AmountFloat()
			if err := accRepo.Update(ctx, tx.AccountID, dto.AccountUpdate{Balance: &f64Balance}); err != nil {
				logger.Error("CompletedHandler: failed to update account balance", "error", err)
				return err
			}
			logger.Info("CompletedHandler: account balance updated", "account_id", acc.ID, "new_balance", f64Balance)
			return nil
		})
		if err != nil {
			logger.Error("CompletedHandler: transaction failed", "error", err)
			return
		}
		// Optionally: publish a UI/account balance update event
		// _ = bus.Publish(ctx, events.AccountBalanceUpdatedEvent{UserID: tx.UserID, AccountID: tx.AccountID, NewBalance: ...})
	}
}
