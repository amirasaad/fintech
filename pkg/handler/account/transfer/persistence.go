package transfer

import (
	"context"
	"log/slog"
	"time"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
)

// TransferPersistenceHandler handles TransferDomainOpDoneEvent, persists to DB, and publishes TransferPersistedEvent.
func TransferPersistenceHandler(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "TransferPersistenceHandler", "event_type", e.Type())
		log.Info("🟢 [START] Received event", "event", e)
		evt, ok := e.(events.TransferDomainOpDoneEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event type", "event", e)
			return nil
		}
		log.Info("🔄 [PROCESS] Received TransferDomainOpDoneEvent, persisting transfer",
			"event", evt,
			"dest_account_id", evt.DestAccountID,
			"source_account_id", evt.AccountID,
			"sender_user_id", evt.UserID,
			"receiver_user_id", evt.ReceiverUserID)

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
				UserID:    evt.UserID,
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
			return nil
		}
		log.Info("📤 [EMIT] Emitting TransferPersistedEvent", "dest_account_id", evt.DestAccountID)
		correlationID := evt.CorrelationID
		persistedEvent := events.TransferPersistedEvent{
			TransferDomainOpDoneEvent: evt,
			// Add any additional fields as needed
		}
		log.Info("📤 [EMIT] Emitting TransferPersistedEvent", "event", persistedEvent, "correlation_id", correlationID.String())
		if err := bus.Emit(ctx, persistedEvent); err != nil {
			return err
		}

		txID := uuid.New() // Assuming txID is available from the transaction creation
		ve := evt          // Assuming evt is the validated event

		log.Info("DEBUG: ve.DestAccountID, ve.ReceiverUserID", "dest_account_id", ve.DestAccountID, "receiver_user_id", ve.ReceiverUserID)
		log.Info("source_account_id", "account_id", evt.AccountID, "sender_user_id", evt.UserID)
		conversionEvent := events.ConversionRequestedEvent{
			FlowEvent:  evt.FlowEvent,
			ID:         uuid.New(),
			FromAmount: ve.Amount,
			ToCurrency: ve.Amount.Currency().String(),
			RequestID:  txID.String(),
			Timestamp:  time.Now(),
		}
		log.Info("DEBUG: Full ConversionRequestedEvent", "event", conversionEvent)
		log.Info("📤 [EMIT] About to emit ConversionRequestedEvent", "handler", "TransferPersistenceHandler", "event_type", conversionEvent.Type(), "correlation_id", correlationID.String())
		return bus.Emit(ctx, conversionEvent)
	}
}
