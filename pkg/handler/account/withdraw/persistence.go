package withdraw

import (
	"context"
	"errors"
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

// WithdrawPersistenceHandler handles WithdrawValidatedEvent: persists the withdraw transaction and emits WithdrawPersistedEvent.
func WithdrawPersistenceHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With("handler", "WithdrawPersistenceHandler", "event_type", e.Type())
		log.Info("🟢 [START] Received event", "event", e)

		ve, ok := e.(events.WithdrawValidatedEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event", "event", e)
			return
		}
		log.Info("🔄 [PROCESS] Received WithdrawValidatedEvent", "event", ve)

		txID := uuid.New()
		if err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			txRepoAny, err := uow.GetRepository((*transaction.Repository)(nil))
			if err != nil {
				log.Error("❌ [ERROR] Failed to get repo", "err", err)
				return err
			}
			txRepo, ok := txRepoAny.(transaction.Repository)
			if !ok {
				log.Error("❌ [ERROR] Failed to retrieve repo type")
				return errors.New("failed to retrieve repo type")
			}
			if err := txRepo.Create(ctx, dto.TransactionCreate{
				ID:        txID,
				UserID:    ve.UserID,
				AccountID: ve.AccountID,
				Amount:    ve.Amount.Amount(),
				Currency:  ve.Amount.Currency().String(),
				Status:    "created",
			}); err != nil {
				return err
			}
			log.Info("✅ [SUCCESS] Withdraw transaction persisted", "transaction_id", txID)
			return nil
		}); err != nil {
			log.Error("❌ [ERROR] Failed to persist withdraw transaction", "error", err)
			return
		}
		correlationID := ve.CorrelationID
		persistedEvent := events.WithdrawPersistedEvent{
			WithdrawValidatedEvent: ve,
			TransactionID:          txID,
		}
		log.Info("📤 [EMIT] Emitting WithdrawPersistedEvent", "event", persistedEvent, "correlation_id", correlationID.String())
		_ = bus.Publish(ctx, persistedEvent)

		// Emit ConversionRequested to trigger currency conversion for withdraw (decoupled from payment)
		correlationID = uuid.New()
		log.Info("DEBUG: ve.UserID and ve.AccountID", "user_id", ve.UserID, "account_id", ve.AccountID)
		conversionEvent := events.ConversionRequestedEvent{
			FlowEvent:  ve.FlowEvent,
			ID:         uuid.New(),
			FromAmount: ve.Amount,
			ToCurrency: ve.Amount.Currency().String(),
			RequestID:  txID.String(),
			Timestamp:  time.Now(),
		}
		log.Info("DEBUG: Full ConversionRequestedEvent", "event", conversionEvent)
		log.Info("📤 [EMIT] About to emit ConversionRequestedEvent", "handler", "WithdrawPersistenceHandler", "event_type", conversionEvent.Type(), "correlation_id", correlationID.String())
		_ = bus.Publish(ctx, conversionEvent)
	}
}
