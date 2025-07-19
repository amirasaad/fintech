// Package deposit previously contained DepositPersistenceHandler, now moved to pkg/handler/payment/persistence_handler.go
package deposit

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
)

// PersistenceHandler handles DepositValidatedEvent: converts the float64 amount and currency to money.Money, persists the transaction, and emits DepositPersistedEvent.
func PersistenceHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With("handler", "DepositPersistenceHandler", "event_type", e.EventType())
		log.Info("🟢 [START] Received event", "event", e)

		// Expect DepositValidatedEvent from validation handler
		ve, ok := e.(events.DepositValidatedEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event", "event", e)
			return
		}
		log.Info("🔄 [PROCESS] Received DepositValidatedEvent", "event", ve)

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
			log.Info("✅ [SUCCESS] Transaction persisted", "transaction_id", txID)
			return nil
		}); err != nil {
			log.Error("❌ [ERROR] Failed to persist transaction", "error", err)
			return
		}

		// Emit DepositPersistedEvent
		log.Info("📤 [EMIT] Emitting DepositPersistedEvent", "transaction_id", txID)
		_ = bus.Publish(ctx, events.DepositPersistedEvent{
			DepositValidatedEvent: ve,
			TransactionID:         txID,
			UserID:                ve.UserID,
			Amount:                ve.Amount,
		})

		// Emit ConversionRequested to trigger currency conversion for deposit (decoupled from payment)
		log.Info("📤 [EMIT] Emitting ConversionRequested for deposit", "transaction_id", txID)
		_ = bus.Publish(ctx, events.ConversionRequested{
			CorrelationID:  txID.String(),
			FlowType:       "deposit",
			OriginalEvent:  ve,
			Amount:         ve.Amount,
			SourceCurrency: ve.Amount.Currency().String(),
			TargetCurrency: ve.Account.Currency().String(),
		})
	}
}
