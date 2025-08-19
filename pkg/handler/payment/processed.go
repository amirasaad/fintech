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
	"gorm.io/gorm"
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
		log = log.With(
			"transaction_id", pp.TransactionID,
			"payment_id", *pp.PaymentID,
		)
		log.Info(
			"🔄 [PROCESS] Updating transaction with payment ID")

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
			if transactionID == uuid.Nil && pp.PaymentID != nil {
				tx, getErr := txRepo.GetByPaymentID(ctx, *pp.PaymentID)
				if getErr != nil {
					log.Error(
						"Failed to get transaction by payment ID",
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
			// First, try to get the existing transaction
			tx, getErr := txRepo.Get(ctx, transactionID)
			if getErr != nil && !errors.Is(getErr, gorm.ErrRecordNotFound) {
				log.Error(
					"Failed to get transaction",
					"error", getErr,
				)
				return fmt.Errorf("failed to get transaction: %w", getErr)
			}

			// If transaction exists, update it with payment ID
			if tx != nil {
				update := dto.TransactionUpdate{
					PaymentID: pp.PaymentID,
					Status:    &status,
				}
				if err := txRepo.Update(ctx, transactionID, update); err != nil {
					log.Error(
						"Failed to update transaction with payment ID",
						"error", err,
					)
					return fmt.Errorf("failed to update transaction: %w", err)
				}
				log.Info(
					"Updated existing transaction with payment ID",
				)
				return nil
			}

			// If transaction doesn't exist, create a new one
			txCreate := dto.TransactionCreate{
				ID:          transactionID,
				UserID:      pp.UserID,
				AccountID:   pp.AccountID,
				Status:      status,
				MoneySource: "Stripe", // Default money source for Stripe payments
				PaymentID:   pp.PaymentID,
			}

			// Set amount and currency if available
			if pp.Amount != nil {
				txCreate.Amount = int64(pp.Amount.Amount())
				txCreate.Currency = pp.Amount.Currency().String()
			}

			// Create the transaction using UpsertByPaymentID which handles both create and update
			if err := txRepo.UpsertByPaymentID(ctx, *pp.PaymentID, txCreate); err != nil {
				log.Error(
					"Failed to create/update transaction with payment ID",
					"error", err,
				)
				return fmt.Errorf("failed to create/update transaction: %w", err)
			}

			log.Info(
				"Transaction updated with payment ID",
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
