package payment

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
)

// PaymentIdUpdateHandler handles PaymentInitiatedEvent, updates the transaction's payment ID, and emits PaymentIdPersistedEvent.
func PaymentIdUpdateHandler(bus eventbus.EventBus, uow repository.UnitOfWork) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		slog.Info("PaymentIdUpdateHandler: received event", "event", e)
		pie, ok := e.(events.PaymentInitiatedEvent)
		if !ok {
			return
		}
		err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			txRepoAny, err := uow.GetRepository((*transaction.Repository)(nil))

			if err != nil {
				slog.Error("PaymentIdUpdateHandler: failed to get transaction repo", "error", err)
				return err
			}
			txRepo, ok := txRepoAny.(transaction.Repository)
			if !ok {
				return errors.New("failed to retrieve transaction repo type")
			}
			update := dto.TransactionUpdate{
				PaymentID: &pie.PaymentID,
			}
			if err := txRepo.Update(ctx, pie.TransactionID, update); err != nil {
				slog.Error("PaymentIdUpdateHandler: failed to update payment ID", "error", err)
				return err
			}
			_ = bus.Publish(ctx, events.PaymentIdPersistedEvent{
				PaymentInitiatedEvent: pie,
				TransactionID:         pie.TransactionID,
			})
			return nil
		})
		if err != nil {
			slog.Error("PaymentIdUpdateHandler: persistence failed", "error", err)
			return
		}
	}
}
