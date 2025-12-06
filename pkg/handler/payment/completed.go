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
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

// ExtractPaymentCompletedKey extracts idempotency key from PaymentCompleted event
func ExtractPaymentCompletedKey(e events.Event) string {
	pc, ok := e.(*events.PaymentCompleted)
	if !ok {
		return ""
	}
	if pc.PaymentID != nil && *pc.PaymentID != "" {
		return *pc.PaymentID
	}
	if pc.TransactionID != uuid.Nil {
		return pc.TransactionID.String()
	}
	return ""
}

// HandleCompleted handles PaymentCompletedEvent,
// updates the transaction status in the DB, and publishes a follow-up event if needed.
func HandleCompleted(
	bus eventbus.Bus,
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
		if logger == nil {
			logger = slog.Default()
		}
		log := logger.With(
			"handler", "payment.HandleCompleted",
			"event", e,
			"event_type", e.Type(),
		)
		log.Info(
			"🟢 [START] HandleCompleted received event",
		)
		log.Debug("🟢 Handling PaymentCompleted event",
			"event_type", e.Type(),
			"event", fmt.Sprintf("%+v", e),
		)
		pc, ok := e.(*events.PaymentCompleted)
		if !ok {
			log.Error(
				"Skipping unexpected event type",
				"event", e,
			)
			return nil
		}
		log = log.With(
			"user_id", pc.UserID,
			"account_id", pc.AccountID,
			"payment_id", *pc.PaymentID,
			"transaction_id", pc.TransactionID,
		)

		if err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			accRepo, err := common.GetAccountRepository(uow, log)
			if err != nil {
				return err
			}
			txRepo, err := common.GetTransactionRepository(uow, log)
			if err != nil {
				log.Error("failed to get transaction repository", "error", err)
				return err
			}
			if pc.PaymentID == nil {
				log.Error("payment ID is nil")
				return fmt.Errorf("payment ID is nil")
			}

			// Lookup transaction by payment ID or transaction ID
			lookupResult := common.LookupTransactionByPaymentOrID(
				ctx,
				txRepo,
				pc.PaymentID,
				pc.TransactionID,
				log,
			)

			if lookupResult.Error != nil {
				return lookupResult.Error
			}

			if !lookupResult.Found {
				return nil // Skip gracefully if transaction not found
			}

			tx := lookupResult.Transaction

			// Update the transaction with the payment ID if it wasn't set
			if tx.PaymentID == nil || (tx.PaymentID != nil && *tx.PaymentID != *pc.PaymentID) {
				update := dto.TransactionUpdate{
					PaymentID: pc.PaymentID,
				}
				if uerr := txRepo.Update(ctx, tx.ID, update); uerr != nil {
					log.Error(
						"failed to update transaction with payment ID",
						"transaction_id", tx.ID,
						"payment_id", pc.PaymentID,
						"error", uerr,
					)
					return fmt.Errorf("failed to update transaction: %w", uerr)
				}
				tx.PaymentID = pc.PaymentID
			}

			log = log.With(
				"transaction_id", tx.ID,
				"user_id", tx.UserID,
			)
			acc, err := accRepo.Get(ctx, tx.AccountID)
			if err != nil {
				log.Error(
					"failed to get account",
					"error", err,
				)
				return err
			}
			domainAcc, err := mapper.MapAccountReadToDomain(acc)
			if err != nil {
				log.Error(
					"failed to map account to domain",
					"error", err,
				)
				return err
			}

			// Log provider fee details before calculation
			newBalance, err := domainAcc.Balance.Add(pc.Amount)
			if err != nil {
				log.Error(
					"failed to add net transaction amount to balance",
					"error", err,
				)
				return err
			}
			oldStatus := tx.Status
			status := string(account.TransactionStatusCompleted)
			tx.Status = status

			// Store the gross amount in the transaction
			amount := pc.Amount.Amount()
			currency := pc.Amount.Currency().String()
			balance := newBalance.Amount()

			update := dto.TransactionUpdate{
				Status:   &status,
				Amount:   &amount,
				Currency: &currency,
				Balance:  &balance,
			}

			if err = txRepo.Update(ctx, tx.ID, update); err != nil {
				log.Error(
					"failed to update transaction status",
					"error", err,
				)
				return err
			}

			log.Info(
				"✅ [SUCCESS] transaction status updated",
				"old_status", oldStatus,
				"new_status", tx.Status,
			)

			f64Balance := newBalance.Amount()
			if err := accRepo.Update(
				ctx,
				tx.AccountID,
				dto.AccountUpdate{Balance: &f64Balance},
			); err != nil {
				log.Error(
					"failed to update account balance",
					"error", err,
				)
				return err
			}

			log.Info(
				"✅ [SUCCESS] account balance updated",
				"account_id", acc.ID,
				"new_balance", newBalance,
				"balance", domainAcc.Balance,
			)

			log.Info(
				"✅ [SUCCESS] emitted FeesCalculated event",
				"transaction_id", tx.ID)
			return nil
		}); err != nil {
			log.Error(
				"uow.Do failed",
				"error", err,
			)
			return err
		}
		return nil
	}
}
