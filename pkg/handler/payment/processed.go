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

// ExtractPaymentProcessedKey extracts idempotency key from PaymentProcessed event
func ExtractPaymentProcessedKey(e events.Event) string {
	pp, ok := e.(*events.PaymentProcessed)
	if !ok {
		return ""
	}
	if pp.PaymentID != nil && *pp.PaymentID != "" {
		return *pp.PaymentID
	}
	if pp.TransactionID != uuid.Nil {
		return pp.TransactionID.String()
	}
	return ""
}

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
		// Build log fields safely without dereferencing nil pointers
		logFields := []any{
			"transaction_id", pp.TransactionID,
		}
		if pp.PaymentID != nil {
			logFields = append(logFields, "payment_id", *pp.PaymentID)
		}
		log = log.With(logFields...)
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

			// Lookup transaction by payment ID or transaction ID
			lookupResult := common.LookupTransactionByPaymentOrID(
				ctx,
				txRepo,
				pp.PaymentID,
				pp.TransactionID,
				log,
			)

			if lookupResult.Error != nil {
				return lookupResult.Error
			}

			if !lookupResult.Found {
				return nil // Skip gracefully if transaction not found
			}

			tx := lookupResult.Transaction
			transactionID := lookupResult.TransactionID
			status := "processed"

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
