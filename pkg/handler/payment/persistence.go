package payment

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
)

// Persistence handles PaymentInitiatedEvent and updates the transaction with payment ID.
// This is a generic handler that can process payment events for all operations (deposit, withdraw, transfer).
func Persistence(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e common.Event) error {
	return func(ctx context.Context, e common.Event) error {
		log := logger.With(
			"handler", "Persistence",
			"event_type", e.Type(),
		)
		log.Info("🟢 [START] Received event", "event", e)

		pie, ok := e.(*events.PaymentInitiatedEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event type for payment persistence", "event", e)
			return nil
		}

		log.Info("🔄 [PROCESS] Updating transaction with payment ID",
			"transaction_id", pie.TransactionID,
			"payment_id", pie.PaymentID)

		// Update the transaction with payment ID
		if err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			txRepoAny, err := uow.GetRepository((*transaction.Repository)(nil))
			if err != nil {
				log.Error("❌ [ERROR] Failed to get transaction repo", "error", err)
				return err
			}
			txRepo, ok := txRepoAny.(transaction.Repository)
			if !ok {
				log.Error("❌ [ERROR] Failed to retrieve repo type")
				return errors.New("failed to retrieve repo type")
			}

			// Check if the transaction already has a payment ID before updating
			tx, err := txRepo.Get(ctx, pie.TransactionID)
			if err != nil {
				log.Error("❌ [ERROR] Failed to get transaction before update", "error", err)
				return err
			}
			if tx.PaymentID != "" {
				log.Warn("🚫 [SKIP] Duplicate PaymentIdPersistedEvent emission detected: transaction already has payment ID", "transaction_id", pie.TransactionID, "existing_payment_id", tx.PaymentID)
				return errors.New("transaction already has payment ID")
			}

			status := string(account.TransactionStatusPending)
			if err = txRepo.Update(ctx, pie.TransactionID, dto.TransactionUpdate{
				PaymentID: &pie.PaymentID,
				Status:    &status,
			}); err != nil {
				log.Error("❌ [ERROR] Failed to update transaction with payment ID", "error", err)
				return err
			}

			log.Info("✅ [SUCCESS] Transaction updated with payment ID",
				"transaction_id", pie.TransactionID,
				"payment_id", pie.PaymentID)

			// Guard: Only emit if TransactionID is valid and no cycle will occur
			if pie.TransactionID == uuid.Nil {
				log.Error("❌ [ERROR] Transaction ID is nil, aborting PaymentIdPersistedEvent emission", "event", pie)
				return errors.New("invalid transaction ID")
			}

			return nil
		}); err != nil {
			log.Error("❌ [ERROR] Payment persistence failed", "error", err)
			return err
		}
		return nil
	}
}
