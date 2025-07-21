// Package deposit previously contained DepositPersistenceHandler, now moved to pkg/handler/payment/persistence_handler.go
package deposit

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

// Persistence handles DepositValidatedEvent: converts the float64 amount and currency to money.Money, persists the transaction, and emits DepositPersistedEvent.
func Persistence(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "Persistence", "event_type", e.Type())
		depth, _ := ctx.Value("eventDepth").(int)
		log.Info("[DEPTH] Event received", "type", e.Type(), "depth", depth, "event", e)
		log.Info("🟢 [START] Received event", "event", e)

		// Expect DepositValidatedEvent from validation handler
		ve, ok := e.(events.DepositValidatedEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event", "event", e)
			return nil
		}
		correlationID := ve.CorrelationID
		if correlationID == uuid.Nil {
			correlationID = uuid.New()
		}
		log = log.With("correlation_id", correlationID)
		log.Info("🔄 [PROCESS] Received DepositValidatedEvent", "event", ve)

		// Log the currency of the validated deposit before persisting
		log.Info("[CHECK] DepositValidatedEvent amount currency before persist", "currency", ve.Amount.Currency().String())
		// ve.Amount should always be the source currency at this stage

		// Create a new transaction and persist it
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
				ID:          txID,
				UserID:      ve.UserID,
				AccountID:   ve.AccountID,
				Amount:      ve.Amount.Amount(),
				Currency:    ve.Amount.Currency().String(),
				Status:      "created",
				MoneySource: ve.Source,
			}); err != nil {
				return err
			}
			log.Info("✅ [SUCCESS] Transaction persisted", "transaction_id", txID, "correlation_id", correlationID)
			return nil
		}); err != nil {
			log.Error("❌ [ERROR] Failed to persist transaction", "error", err)
			return err
		}

		// Emit DepositPersistedEvent
		persistedEvent := events.DepositPersistedEvent{
			DepositValidatedEvent: ve,
			TransactionID:         txID,
			Amount:                ve.Amount,
		}
		log.Info("📤 [EMIT] Emitting DepositPersistedEvent", "event", persistedEvent, "correlation_id", correlationID.String())
		if err := bus.Emit(ctx, persistedEvent); err != nil {
			return err
		}

		// Emit ConversionRequested to trigger currency conversion for deposit (decoupled from payment)
		log.Info("📤 [EMIT] Emitting ConversionRequestedEvent for deposit", "transaction_id", txID, "correlation_id", correlationID)
		log.Info("DEBUG: ve.UserID and ve.AccountID", "user_id", ve.UserID, "account_id", ve.AccountID)
		conversionEvent := events.ConversionRequestedEvent{
			FlowEvent:     ve.FlowEvent,
			ID:            uuid.New(),
			FromAmount:    ve.Amount,
			ToCurrency:    ve.Account.Currency().String(),
			RequestID:     txID.String(),
			TransactionID: txID, // Always propagate!
			Timestamp:     time.Now(),
		}
		log.Info("DEBUG: Full ConversionRequestedEvent", "event", conversionEvent)
		log.Info("📤 [EMIT] About to emit ConversionRequestedEvent", "handler", "Persistence", "event_type", conversionEvent.Type(), "correlation_id", correlationID.String())
		return bus.Emit(ctx, &conversionEvent)
	}
}
