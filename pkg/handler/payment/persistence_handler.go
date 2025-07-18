package payment

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
)

// PersistenceHandler handles MoneyConvertedEvent, persists the transaction, and emits DepositPersistedEvent with TransactionID.
func PersistenceHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		logger := logger.With(
			"handler", "PersistenceHandler",
			"event_type", e.EventType(),
		)
		logger.Info("received event", "event", e)
		pie, ok := e.(events.PaymentInitiatedEvent)
		if !ok {
			logger.Error("PersistenceHandler: unexpected event type", "type", e.EventType(), "event", pie, "error", e)
			return
		}
		err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			txRepoAny, err := uow.GetRepository((*transaction.Repository)(nil))
			if err != nil {
				logger.Error("PersistenceHandler: failed to get transaction repo", "error", err)
				return err
			}
			txRepo, ok := txRepoAny.(transaction.Repository)
			if !ok {
				logger.Error("PersistenceHandler: failed to retrieve repo type")
				return errors.New("failed to retrieve repo type")
			}
			status := account.TransactionStatusPending
			if err := txRepo.Update(ctx, pie.TransactionID, dto.TransactionUpdate{PaymentID: &pie.PaymentID, Status: &status}); err != nil {
				logger.Error("PersistenceHandler: failed to update transaction", "error", err)
				return err
			}
			_ = bus.Publish(ctx, events.PaymentIdPersistedEvent{
				PaymentInitiatedEvent: events.PaymentInitiatedEvent{
					Status:        pie.Status,
					TransactionID: pie.TransactionID,
					UserID:        pie.UserID,
				},
			})
			return nil
		})
		if err != nil {
			logger.Error("PersistenceHandler: persistence failed", "error", err)
			return
		}
	}
}
