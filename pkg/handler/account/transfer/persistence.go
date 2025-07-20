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

// Persistence handles TransferDomainOpDoneEvent, persists to DB, and publishes TransferPersistedEvent.
func Persistence(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "Persistence", "event_type", e.Type())
		log.Info("🟢 [START] Received event", "event", e)
		// Expect TransferDomainOpDoneEvent
		te, ok := e.(events.TransferDomainOpDoneEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event type", "event", e)
			return nil
		}
		log.Info("🔄 [PROCESS] Internal transfer: creating tx_out and tx_in, updating balances",
			"source_account_id", te.AccountID,
			"dest_account_id", te.DestAccountID,
			"amount", te.Amount.Amount(),
			"currency", te.Amount.Currency().String(),
			"sender_user_id", te.UserID,
			"receiver_user_id", te.ReceiverUserID)

		// Atomic operation: create tx_out, tx_in, update balances
		err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
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
			txOutID := uuid.New()
			txInID := uuid.New()
			amount := te.Amount.Amount()
			currency := te.Amount.Currency().String()
			// tx_out: sender, negative amount
			if err := txRepo.Create(ctx, dto.TransactionCreate{
				ID:          txOutID,
				UserID:      te.UserID,
				AccountID:   te.AccountID,
				Amount:      -amount,
				Currency:    currency,
				Status:      "completed",
				MoneySource: "transfer",
			}); err != nil {
				return err
			}
			// tx_in: receiver, positive amount
			if err := txRepo.Create(ctx, dto.TransactionCreate{
				ID:          txInID,
				UserID:      te.ReceiverUserID,
				AccountID:   te.DestAccountID,
				Amount:      amount,
				Currency:    currency,
				Status:      "completed",
				MoneySource: "transfer",
			}); err != nil {
				return err
			}
			// TODO: Update both account balances here (call account repo)
			log.Info("✅ [SUCCESS] Internal transfer transactions created", "tx_out_id", txOutID, "tx_in_id", txInID)
			// Emit TransferCompletedEvent
			completedEvent := events.TransferCompletedEvent{
				TransferDomainOpDoneEvent: te,
				TxOutID:                   txOutID,
				TxInID:                    txInID,
			}
			if err := bus.Emit(ctx, completedEvent); err != nil {
				return err
			}
			// Emit ConversionRequestedEvent for transfer (for currency conversion, if needed)
			conversionEvent := events.ConversionRequestedEvent{
				FlowEvent:     te.FlowEvent,
				ID:            uuid.New(),
				FromAmount:    te.Amount,
				ToCurrency:    te.Amount.Currency().String(),
				RequestID:     txOutID.String(),
				Timestamp:     time.Now(),
				TransactionID: txOutID,
			}
			log.Info("[EMIT] About to emit ConversionRequestedEvent for transfer", "event", conversionEvent)
			if err := bus.Emit(ctx, conversionEvent); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			log.Error("❌ [ERROR] Failed to persist internal transfer", "error", err)
			return err
		}
		log.Info("📤 [EMIT] Emitting TransferCompletedEvent", "source_account_id", te.AccountID, "dest_account_id", te.DestAccountID)
		return nil
	}
}
