package fees

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/handler/common"
	"github.com/amirasaad/fintech/pkg/repository"
)

// HandleCalculated handles FeesCalculated events.
// It updates the transaction with the calculated fees and deducts them from the account balance.
func HandleCalculated(
	uow repository.UnitOfWork,
	logger *slog.Logger,
) eventbus.HandlerFunc {
	return func(
		ctx context.Context,
		e events.Event,
	) error {
		log := logger.With(
			"handler", "fees.HandleCalculated",
			"event_type", e.Type(),
		)
		log.Info("🟢 [START] Processing FeesCalculated event")

		// Type assert to get the FeesCalculated event
		fc, ok := e.(*events.FeesCalculated)
		if !ok {
			err := fmt.Errorf("unexpected event type: %s", e.Type())
			log.Error("unexpected event type", "error", err)
			return err
		}

		log = log.With(
			"transaction_id", fc.TransactionID,
			"event_id", fc.ID,
			"fee_amount", fc.Fee.Amount,
		)

		if err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			// Get transaction repository
			txRepo, err := common.GetTransactionRepository(uow, log)
			if err != nil {
				log.Error(
					"failed to get transaction repository", "error", err)
				return fmt.Errorf("failed to get transaction repository: %w", err)
			}

			// Get account repository
			accRepo, err := common.GetAccountRepository(uow, log)
			if err != nil {
				log.Error(
					"failed to get account repository", "error", err)
				return fmt.Errorf("failed to get account repository: %w", err)
			}

			// Create fee calculator and apply fees
			calculator := NewFeeCalculator(txRepo, accRepo, log)
			if err := calculator.ApplyFees(ctx, fc.TransactionID, fc.Fee); err != nil {
				log.Error("failed to apply fees", "error", err)
				return fmt.Errorf("failed to apply fees: %w", err)
			}

			log.Info("✅ Successfully processed fee calculation")
			return nil
		}); err != nil {
			log.Error("failed to process FeesCalculated event", "error", err)
			return err
		}

		log.Info("🟢 [END] Processing FeesCalculated event")
		return nil
	}
}
