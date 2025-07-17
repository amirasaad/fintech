package payment

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	events "github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
)

// PaymentCompletedHandler handles PaymentCompletedEvent, updates the transaction status in the DB, and publishes a follow-up event if needed.
func PaymentCompletedHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		logger.Info("PaymentCompletedHandler: received event", "event", e)
		pe, ok := e.(*events.PaymentCompletedEvent)
		if !ok {
			logger.Error("PaymentCompletedHandler: unexpected event type", "event", e)
			return
		}
		repo, err := uow.TransactionRepository()
		if err != nil {
			logger.Error("PaymentCompletedHandler: failed to get transaction repo", "error", err)
			return
		}
		tx, err := repo.GetByPaymentID(pe.PaymentID)
		if err != nil {
			logger.Error("PaymentCompletedHandler: failed to get transaction by payment ID", "error", err, "payment_id", pe.PaymentID)
			return
		}
		tx.Status = accountdomain.TransactionStatus(pe.Status)
		if err := repo.Update(tx); err != nil {
			logger.Error("PaymentCompletedHandler: failed to update transaction status", "error", err, "payment_id", pe.PaymentID)
			return
		}
		// Optionally: publish a TransactionStatusUpdatedEvent or similar
		// _ = bus.Publish(ctx, events.TransactionStatusUpdatedEvent{...})
	}
}
