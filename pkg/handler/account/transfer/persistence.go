package transfer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
)

// Persistence handles the final, atomic persistence of a transfer.
func Persistence(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e common.Event) error {
	return func(ctx context.Context, e common.Event) error {
		log := logger.With("handler", "FinalPersistence", "event_type", e.Type())

		// 1. Defensive: Check event type and structure
		te, ok := e.(*events.TransferCompleted)
		if !ok {
			log.Error("❌ [DISCARD] Unexpected event type", "event", e)
			return nil
		}
		tr := te.TransferRequested
		log = log.With("correlation_id", tr.CorrelationID)
		log.Info("🟢 [START] Received event", "event", te)

		if tr.AccountID == uuid.Nil || tr.DestAccountID == uuid.Nil || tr.Amount.IsZero() {
			log.Error("❌ [DISCARD] Malformed final persistence event", "event", te)
			return nil
		}

		// 2. Atomic Final Persistence
		txInID := uuid.New()
		txOutID := tr.TransactionID

		err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			txRepoAny, err := uow.GetRepository((*transaction.Repository)(nil))
			if err != nil {
				return fmt.Errorf("failed to get transaction repo: %w", err)
			}
			txRepo := txRepoAny.(transaction.Repository)

			accRepoAny, err := uow.GetRepository((*account.Repository)(nil))
			if err != nil {
				return fmt.Errorf("failed to get account repo: %w", err)
			}
			accRepo := accRepoAny.(account.Repository)

			// Don't update status here, we'll update it based on success/failure after all operations

			// c. Atomically update account balances using money value object
			sourceAcc, err := accRepo.Get(ctx, te.TransferRequested.AccountID)
			if err != nil {
				return fmt.Errorf("could not find source account: %w", err)
			}
			destAcc, err := accRepo.Get(ctx, tr.DestAccountID)
			if err != nil {
				return fmt.Errorf("could not find destination account: %w", err)
			}

			sourceBalance, err := money.New(sourceAcc.Balance, tr.Amount.Currency())
			if err != nil {
				return fmt.Errorf("could not create money for source balance: %w", err)
			}
			destBalance, err := money.New(destAcc.Balance, tr.Amount.Currency())
			if err != nil {
				return fmt.Errorf("could not create money for dest balance: %w", err)
			}

			newSourceMoney, err := sourceBalance.Subtract(tr.Amount)
			if err != nil {
				return fmt.Errorf("could not subtract from source balance: %w", err)
			}
			newDestMoney, err := destBalance.Add(tr.Amount)
			if err != nil {
				return fmt.Errorf("could not add to dest balance: %w", err)
			}

			newSourceBalance := newSourceMoney.AmountFloat()
			newDestBalance := newDestMoney.AmountFloat()

			if err := accRepo.Update(ctx, tr.AccountID, dto.AccountUpdate{Balance: &newSourceBalance}); err != nil {
				return fmt.Errorf("failed to debit source account: %w", err)
			}
			if err := accRepo.Update(ctx, tr.DestAccountID, dto.AccountUpdate{Balance: &newDestBalance}); err != nil {
				return fmt.Errorf("failed to credit destination account: %w", err)
			}

			// Only mark as completed if all operations succeeded
			completedStatus := "completed"
			if err := txRepo.Update(ctx, txOutID, dto.TransactionUpdate{Status: &completedStatus}); err != nil {
				return fmt.Errorf("failed to update transaction status to completed: %w", err)
			}

			return nil
		})

		if err != nil {
			log.Error("❌ [ERROR] Final persistence transaction failed", "error", err)
			// Create the failure event first so we can use the error message
			failureEvent := events.TransferFailed{
				TransferRequested: te.TransferRequested,
				Reason:            "PersistenceFailed: " + err.Error(),
			}

			// Try to mark the transaction as failed, but don't let this fail the whole operation
			err2 := uow.Do(ctx, func(uow repository.UnitOfWork) error {
				txRepoAny, err := uow.GetRepository((*transaction.Repository)(nil))
				if err != nil {
					return fmt.Errorf("failed to get transaction repo: %w", err)
				}
				txRepo := txRepoAny.(transaction.Repository)

				failedStatus := "failed"
				if updateErr := txRepo.Update(ctx, txOutID, dto.TransactionUpdate{Status: &failedStatus}); updateErr != nil {
					log.Error("❌ [ERROR] Failed to update transaction status to failed", "error", updateErr)
				}
				return nil
			})

			if err2 != nil {
				log.Error("❌ [ERROR] Failed to mark transaction as failed", "error", err2)
			}

			// Emit the failure event regardless of whether marking as failed succeeded
			return bus.Emit(ctx, failureEvent)
		}
		log.Info("✅ [SUCCESS] Final transfer persistence complete", "tx_out_id", txOutID, "tx_in_id", txInID)

		// 4. Emit completion event
		completedEvent := events.NewTransferCompleted(&te.TransferRequested)
		log.Info("📤 [EMIT] Emitting TransferCompletedEvent")
		return bus.Emit(ctx, completedEvent)
	}
}
