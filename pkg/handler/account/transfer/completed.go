package transfer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
)

// HandleCompleted handles the final, atomic persistence of a transfer.
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
		log := logger.With(
			"handler", "transfer.HandleCompleted",
			"event_type", e.Type(),
		)

		// 1. Defensive: Check event type and structure
		te, ok := e.(*events.TransferCompleted)
		if !ok {
			log.Error(
				"❌ [DISCARD] Unexpected event type",
				"event", e,
			)
			return fmt.Errorf("unexpected event type: %T", e)
		}
		tr, ok := te.OriginalRequest.(*events.TransferRequested)
		if !ok {
			log.Error(
				"❌ [DISCARD] Unexpected original request type",
				"event", te,
			)
			return fmt.Errorf("unexpected original request type: %T", te)
		}
		log = log.With("correlation_id", tr.CorrelationID)
		log.Info(
			"🟢 [START] Received event",
			"event", te,
		)

		if tr.AccountID == uuid.Nil || tr.DestAccountID == uuid.Nil || tr.Amount.IsZero() {
			log.Error(
				"❌ [DISCARD] Malformed final persistence event",
				"event", te,
			)
			return fmt.Errorf("malformed final persistence event: %v", te)
		}

		// 2. Atomic Final HandleCompleted
		txInID := uuid.New()
		txOutID := tr.TransactionID

		if err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
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

			sourceAcc, err := accRepo.Get(ctx, tr.AccountID)
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

			newSourceBalance := newSourceMoney.Amount()
			newDestBalance := newDestMoney.Amount()

			if err := accRepo.Update(
				ctx,
				tr.AccountID,
				dto.AccountUpdate{Balance: &newSourceBalance},
			); err != nil {
				return fmt.Errorf("failed to debit source account: %w", err)
			}
			if err := accRepo.Update(
				ctx,
				tr.DestAccountID,
				dto.AccountUpdate{Balance: &newDestBalance},
			); err != nil {
				return fmt.Errorf("failed to credit destination account: %w", err)
			}

			completedStatus := "completed"
			if err := txRepo.Update(
				ctx,
				txOutID,
				dto.TransactionUpdate{Status: &completedStatus},
			); err != nil {
				return fmt.Errorf(
					"failed to update transaction status to completed: %w", err,
				)
			}
			return nil
		}); err != nil {
			log.Error(
				"❌ [ERROR] Final persistence transaction failed",
				"error", err,
			)
			tf := events.NewTransferFailed(tr, "PersistenceFailed: "+err.Error())
			return bus.Emit(ctx, tf)
		}
		log.Info(
			"✅ [SUCCESS] Final transfer persistence complete",
			"tx_out_id", txOutID,
			"tx_in_id", txInID,
		)

		tc := events.NewTransferCompleted(tr)
		log.Info(
			"📤 [EMIT] Emitting",
			"event", tc.Type(),
		)
		return bus.Emit(ctx, tc)
	}
}
