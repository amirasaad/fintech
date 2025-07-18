package transfer

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
)

// TransferPersistenceHandler handles TransferDomainOpDoneEvent, persists to DB, and publishes TransferPersistedEvent.
func TransferPersistenceHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		logger = logger.With("handler", "TransferPersistenceHandler")
		evt, ok := e.(events.TransferDomainOpDoneEvent)
		if !ok {
			logger.Error("unexpected event type", "event", e)
			return
		}
		logger.Info("received TransferDomainOpDoneEvent, persisting transfer",
			"event", evt,
			"dest_account_id", evt.DestAccountID,
			"source_account_id", evt.SourceAccountID,
			"sender_user_id", evt.SenderUserID)

		err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			repoAny, err := uow.GetRepository((*transaction.Repository)(nil))
			if err != nil {
				return err
			}
			txRepo := repoAny.(transaction.Repository)

			// Update the original transaction to completed status (it already has conversion data)
			originalTxID := evt.EventID // This is the original transaction ID from InitialPersistenceHandler
			completedStatus := "completed"
			update := dto.TransactionUpdate{
				Status: &completedStatus,
			}
			if err := txRepo.Update(ctx, originalTxID, update); err != nil {
				logger.Error("failed to update original transaction to completed", "error", err, "transaction_id", originalTxID)
				return err
			}
			logger.Info("original transaction updated to completed status", "transaction_id", originalTxID)

			// Create incoming transaction (positive amount to destination account)
			incomingTxID := uuid.New()
			incomingCreate := dto.TransactionCreate{
				ID:          incomingTxID,
				UserID:      evt.SenderUserID, // Same user for now (internal transfer)
				AccountID:   evt.DestAccountID,
				Amount:      evt.Amount.Amount(), // Positive amount for incoming
				Status:      "completed",         // Transfer is completed
				Currency:    evt.Amount.Currency().String(),
				MoneySource: evt.Source,
			}
			if err := txRepo.Create(ctx, incomingCreate); err != nil {
				logger.Error("failed to create incoming transaction", "error", err)
				return err
			}
			logger.Info("incoming transaction created", "transaction_id", incomingTxID, "amount", evt.Amount.Amount())

			return nil
		})
		if err != nil {
			logger.Error("persistence failed", "error", err)
			return
		}
		logger.Info("transfer persisted successfully - both outgoing and incoming transactions created", "event", evt)
		_ = bus.Publish(ctx, events.TransferPersistedEvent{TransferDomainOpDoneEvent: evt})
	}
}
