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
		log := logger.With("handler", "TransferPersistenceHandler", "event_type", e.EventType())
		log.Info("🟢 [START] Received event", "event", e)
		evt, ok := e.(events.TransferDomainOpDoneEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event type", "event", e)
			return
		}
		log.Info("🔄 [PROCESS] Received TransferDomainOpDoneEvent, persisting transfer",
			"event", evt,
			"dest_account_id", evt.DestAccountID,
			"source_account_id", evt.SourceAccountID,
			"sender_user_id", evt.SenderUserID)

		if err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			txRepoAny, err := uow.GetRepository((*transaction.Repository)(nil))
			if err != nil {
				log.Error("❌ [ERROR] Failed to get repo", "err", err)
				return err
			}
			txRepo, ok := txRepoAny.(transaction.Repository)
			if !ok {
				log.Error("❌ [ERROR] Failed to retrieve repo type")
				return err
			}
			if err := txRepo.Create(ctx, dto.TransactionCreate{
				ID:        uuid.New(),
				UserID:    evt.SenderUserID,
				AccountID: evt.DestAccountID,
				Amount:    evt.Amount.Amount(),
				Currency:  evt.Amount.Currency().String(),
				Status:    "completed",
			}); err != nil {
				return err
			}
			log.Info("✅ [SUCCESS] Incoming transfer transaction created", "dest_account_id", evt.DestAccountID)
			return nil
		}); err != nil {
			log.Error("❌ [ERROR] Failed to create incoming transfer transaction", "error", err)
			return
		}
		log.Info("📤 [EMIT] Emitting TransferPersistedEvent", "dest_account_id", evt.DestAccountID)
		_ = bus.Publish(ctx, events.TransferPersistedEvent{TransferDomainOpDoneEvent: evt})
	}
}
