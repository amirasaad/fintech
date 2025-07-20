package payment

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
)

// PaymentPersistenceHandler handles PaymentInitiatedEvent and updates the transaction with payment ID.
// This is a generic handler that can process payment events for all operations (deposit, withdraw, transfer).
func PaymentPersistenceHandler(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		logger := logger.With(
			"handler", "PaymentPersistenceHandler",
			"event_type", e.Type(),
		)
		logger.Info("received payment initiated event", "event", e)

		pie, ok := e.(events.PaymentInitiatedEvent)
		if !ok {
			logger.Error("unexpected event type for payment persistence", "event", e)
			return nil
		}

		logger.Info("updating transaction with payment ID",
			"transaction_id", pie.TransactionID,
			"payment_id", pie.PaymentID)

		// Update the transaction with payment ID
		if err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			txRepoAny, err := uow.GetRepository((*transaction.Repository)(nil))
			if err != nil {
				logger.Error("failed to get transaction repo", "error", err)
				return err
			}
			txRepo, ok := txRepoAny.(transaction.Repository)
			if !ok {
				logger.Error("failed to retrieve repo type")
				return errors.New("failed to retrieve repo type")
			}

			// Check if the transaction already has a payment ID before updating
			tx, err := txRepo.Get(ctx, pie.TransactionID)
			if err != nil {
				logger.Error("failed to get transaction before update", "error", err)
				return err
			}
			if tx.PaymentID != "" {
				logger.Warn("Duplicate PaymentIdPersistedEvent emission detected: transaction already has payment ID", "transaction_id", pie.TransactionID, "existing_payment_id", tx.PaymentID)
			}

			status := account.TransactionStatusPending
			if err := txRepo.Update(ctx, pie.TransactionID, dto.TransactionUpdate{
				PaymentID: &pie.PaymentID,
				Status:    &status,
			}); err != nil {
				logger.Error("failed to update transaction with payment ID", "error", err)
				return err
			}

			logger.Info("transaction updated with payment ID",
				"transaction_id", pie.TransactionID,
				"payment_id", pie.PaymentID)

			// Guard: Only emit if TransactionID is valid and no cycle will occur
			if pie.TransactionID == uuid.Nil {
				logger.Error("Transaction ID is nil, aborting PaymentIdPersistedEvent emission", "event", pie)
				return errors.New("invalid transaction ID")
			}

			return nil
		}); err != nil {
			logger.Error("payment persistence failed", "error", err)
			return nil
		}
		return nil
	}
}
