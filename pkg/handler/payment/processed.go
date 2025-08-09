package payment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/handler/common"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

// HandleProcessed handles PaymentInitiatedEvent and updates the transaction with payment ID.
// This is a generic handler that can process payment events
// for all operations (deposit, withdraw, transfer).
func HandleProcessed(
	uow repository.UnitOfWork,
	logger *slog.Logger,
) eventbus.HandlerFunc {
	return func(
		ctx context.Context,
		e events.Event,
	) error {
		log := logger.With(
			"handler", "HandleProcessed",
			"event_type", e.Type(),
		)
		log.Info("🟢 [START] event received")

		pp, ok := e.(*events.PaymentProcessed)
		if !ok {
			log.Error(
				"Unexpected event type for payment processed",
				"event", e,
			)
			return errors.New("unexpected event type")
		}

		log.Info(
			"🔄 [PROCESS] Updating transaction with payment ID",
			"transaction_id", pp.TransactionID,
			"payment_id", pp.PaymentID)

		// Update the transaction with payment ID
		err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			txRepo, err := common.GetTransactionRepository(uow, log)
			if err != nil {
				log.Error(
					"Failed to get transaction repo",
					"error", err,
				)
				return fmt.Errorf("failed to get transaction repo: %w", err)
			}

			transactionID := pp.TransactionID
			if transactionID == uuid.Nil {
				tx, getErr := txRepo.GetByPaymentID(ctx, pp.PaymentID)
				if getErr != nil {
					log.Error(
						"Failed to get transaction by payment ID",
						"payment_id", pp.PaymentID,
						"error", getErr,
					)
					return fmt.Errorf("failed to get transaction by payment ID: %w", getErr)
				}
				transactionID = tx.ID
			}

			if transactionID == uuid.Nil {
				err = fmt.Errorf("no transaction ID provided and could not find by payment ID")
				log.Error(
					"Failed to get transaction before update",
					"error", err,
				)
				return err
			}

			status := "processed"
			if err = txRepo.Update(ctx, transactionID, dto.TransactionUpdate{
				Status:    &status,
				PaymentID: &pp.PaymentID,
			}); err != nil {
				log.Error(
					"Failed to update transaction with payment ID",
					"transaction_id", transactionID,
					"payment_id", pp.PaymentID,
					"error", err,
				)
				return fmt.Errorf("failed to update transaction: %w", err)
			}

			log.Info(
				"Transaction updated with payment ID",
				"transaction_id", transactionID,
				"payment_id", pp.PaymentID,
			)
			return nil
		})

		if err != nil {
			log.Error(
				"Uow.Do failed",
				"error", err,
			)
			return err
		}
		log.Info("✅ [SUCCESS] event processed")
		return nil
	}
}
