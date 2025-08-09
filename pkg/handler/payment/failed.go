package payment

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/handler/common"
	"github.com/amirasaad/fintech/pkg/repository"
)

// HandleFailed handles the PaymentFailedEvent by updating the transaction status to "failed"
func HandleFailed(
	bus eventbus.Bus,
	uow repository.UnitOfWork,
	logger *slog.Logger,
) eventbus.HandlerFunc {
	return func(ctx context.Context, event events.Event) error {
		log := logger.With("handler", "payment.HandleFailed", "event_type", event.Type())
		log.Info("handling payment failed event")

		// Check if the event is a PaymentFailed event
		pf, ok := event.(*events.PaymentFailed)
		if !ok {
			err := fmt.Errorf("expected PaymentFailed event, got %T", event)
			log.Error("invalid event type", "error", err)
			return err
		}

		// Use the transaction ID from the event
		txID := pf.TransactionID
		log = log.With("transaction_id", txID, "payment_id", pf.PaymentID)

		// Get the transaction repository
		txRepo, err := common.GetTransactionRepository(uow, log)
		if err != nil {
			err = fmt.Errorf("failed to get transaction repository: %w", err)
			log.Error("repository error", "error", err)
			return err
		}

		// Update the transaction status to failed
		status := string(account.TransactionStatusFailed)
		updateErr := txRepo.Update(ctx, txID, dto.TransactionUpdate{
			PaymentID: &pf.PaymentID,
			Status:    &status,
		})

		if updateErr != nil {
			err = fmt.Errorf("failed to update transaction status: %w", updateErr)
			log.Error("update error", "error", err)
			return err
		}

		// Commit the transaction
		if err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			log.Info("committing transaction update")
			return nil
		}); err != nil {
			err = fmt.Errorf("failed to commit transaction: %w", err)
			log.Error("commit error", "error", err)
			return err
		}

		log.Info("successfully processed payment failed event")
		return nil
	}
}
