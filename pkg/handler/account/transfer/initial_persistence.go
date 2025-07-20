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

// InitialPersistence handles TransferValidatedEvent: creates initial transaction record and triggers conversion.
func InitialPersistence(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "InitialPersistence", "event_type", e.Type())
		log.Info("🟢 [START] Received event", "event", e)

		ve, ok := e.(events.TransferValidatedEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event type", "event", e)
			return nil
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
			return nil
		}

		// For transfer, we need to determine the target currency from the destination account
		// For now, we'll use the same currency as the source (no conversion needed)
		targetCurrency := ve.Amount.Currency().String()

		log.Info("[LOG] About to emit ConversionRequestedEvent for transfer", "transaction_id", txID)
		result := bus.Emit(ctx, events.ConversionRequestedEvent{
			FromAmount:    ve.Amount,
			ToCurrency:    targetCurrency,
			RequestID:     txID.String(),
			TransactionID: txID,
		})
		log.Info("[LOG] ConversionRequestedEvent emitted for transfer", "transaction_id", txID, "result", result)
		return result
	}
}
