package payment

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
)

// HandleProcessed handles PaymentInitiatedEvent and updates the transaction with payment ID.
// This is a generic handler that can process payment events
// for all operations (deposit, withdraw, transfer).
func HandleProcessed(
	uow repository.UnitOfWork,
	logger *slog.Logger,
) func(
	ctx context.Context,
	e events.Event,
) error {
	return func(
		ctx context.Context,
		e events.Event,
	) error {
		log := logger.With(
			"handler", "HandleProcessed",
			"event_type", e.Type(),
		)
		log.Info("🟢 [START] Received event", "event", e)

		pp, ok := e.(*events.PaymentProcessed)
		if !ok {
			log.Error(
				"❌ [ERROR] Unexpected event type for payment persistence",
				"event", e,
			)
			return errors.New("unexpected event type")
		}

		log.Info(
			"🔄 [PROCESS] Updating transaction with payment ID",
			"transaction_id", pp.TransactionID,
			"payment_id", pp.PaymentID)

		// Update the transaction with payment ID
		if err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			txRepoAny, err := uow.GetRepository((*transaction.Repository)(nil))
			if err != nil {
				log.Error(
					"❌ [ERROR] Failed to get transaction repo",
					"error", err,
				)
				return err
			}
			txRepo, ok := txRepoAny.(transaction.Repository)
			if !ok {
				log.Error(
					"❌ [ERROR] Failed to retrieve repo type",
				)
				return errors.New("failed to retrieve repo type")
			}

			// Check if the transaction already has a payment ID before updating
			tx, err := txRepo.Get(ctx, pp.TransactionID)
			if err != nil {
				log.Error(
					"❌ [ERROR] Failed to get transaction before update",
					"error", err,
				)
				return err
			}
			if tx.PaymentID != "" {
				log.Warn(
					"🚫 [SKIP] duplicate emission detected: transaction already has payment ID",
					"transaction_id", pp.TransactionID,
					"existing_payment_id", tx.PaymentID,
				)
				return errors.New("transaction already has payment ID")
			}

			status := string(account.TransactionStatusPending)
			if err = txRepo.Update(ctx, pp.TransactionID, dto.TransactionUpdate{
				PaymentID: &pp.PaymentID,
				Status:    &status,
			}); err != nil {
				log.Error(
					"❌ [ERROR] Failed to update transaction with payment ID",
					"error", err,
				)
				return err
			}

			log.Info(
				"✅ [SUCCESS] Transaction updated with payment ID",
				"transaction_id", pp.TransactionID,
				"payment_id", pp.PaymentID,
			)

			// Guard: Only emit if TransactionID is valid and no cycle will occur
			if pp.TransactionID == uuid.Nil {
				log.Error(
					"❌ [ERROR] Transaction ID is nil, aborting emission",
					"event", pp,
				)
				return errors.New("invalid transaction ID")
			}

			return nil
		}); err != nil {
			log.Error(
				"❌ [ERROR] Payment persistence failed",
				"error", err,
			)
			return err
		}
		return nil
	}
}
