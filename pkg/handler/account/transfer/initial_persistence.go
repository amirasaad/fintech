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

// InitialPersistenceHandler handles TransferValidatedEvent: creates initial transaction record and triggers conversion.
func InitialPersistenceHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With("handler", "TransferInitialPersistenceHandler", "event_type", e.EventType())
		log.Info("🟢 [START] Received event", "event", e)

		ve, ok := e.(events.TransferValidatedEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event type", "event", e)
			return
		}
		log.Info("🔄 [PROCESS] Received TransferValidatedEvent", "event", ve)

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
				return err
			}
			if err := txRepo.Create(ctx, dto.TransactionCreate{
				ID:          txID,
				UserID:      ve.UserID,
				AccountID:   ve.AccountID,
				Amount:      -ve.Amount.Amount(), // Negative amount for outgoing transaction
				Currency:    ve.Amount.Currency().String(),
				Status:      "created",
				MoneySource: "transfer",
			}); err != nil {
				return err
			}
			log.Info("✅ [SUCCESS] Outgoing transfer transaction created", "transaction_id", txID, "amount", -ve.Amount.Amount())
			return nil
		}); err != nil {
			log.Error("❌ [ERROR] Failed to create transfer transaction", "error", err)
			return
		}

		// For transfer, we need to determine the target currency from the destination account
		// For now, we'll use the same currency as the source (no conversion needed)
		targetCurrency := ve.Amount.Currency().String()

		log.Info("📤 [EMIT] Emitting ConversionRequestedEvent for transfer", "transaction_id", txID)
		_ = bus.Publish(ctx, events.ConversionRequestedEvent{
			FromAmount: ve.Amount,
			ToCurrency: targetCurrency,
			RequestID:  txID.String(),
		})
	}
}
