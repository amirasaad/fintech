package deposit

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

// PaymentPersistenceHandler handles PaymentInitiatedEvent and updates the transaction with payment ID.
func PaymentPersistenceHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		logger := logger.With("handler", "DepositPaymentPersistenceHandler")
		logger.Info("received event", "event", e)

		var paymentID string
		var transactionID string

		switch evt := e.(type) {
		case events.PaymentInitiatedEvent:
			paymentID = evt.PaymentID
			transactionID = evt.TransactionID.String()
			logger.Info("received PaymentInitiatedEvent", "event", evt)
		default:
			logger.Error("unexpected event type for deposit payment persistence", "event", e)
			return
		}

		// Parse transaction ID
		txID, err := uuid.Parse(transactionID)
		if err != nil {
			logger.Error("invalid transaction ID", "transaction_id", transactionID, "error", err)
			return
		}

		logger.Info("updating transaction with payment ID",
			"transaction_id", txID,
			"payment_id", paymentID)

		// Update the transaction with payment ID
		err = uow.Do(ctx, func(uow repository.UnitOfWork) error {
			txRepoAny, err := uow.GetRepository((*transaction.Repository)(nil))
			if err != nil {
				logger.Error("failed to get transaction repository", "error", err)
				return err
			}
			txRepo, ok := txRepoAny.(transaction.Repository)
			if !ok {
				return err
			}

			// Update transaction with payment ID
			update := dto.TransactionUpdate{
				PaymentID: &paymentID,
				Status:    &[]string{"initiated"}[0], // Update status to initiated
			}

			if err := txRepo.Update(ctx, txID, update); err != nil {
				logger.Error("failed to update transaction with payment ID", "error", err)
				return err
			}

			logger.Info("transaction updated with payment ID", "transaction_id", txID)
			return nil
		})

		if err != nil {
			logger.Error("failed to persist payment ID", "error", err)
			return
		}

		logger.Info("payment ID persisted successfully", "transaction_id", txID)
	}
}
