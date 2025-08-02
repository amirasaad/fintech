package transaction

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
)

// ConversionPersistence handles persisting conversion details for any transaction.
func ConversionPersistence(uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e common.Event) error {
	return func(ctx context.Context, e common.Event) error {
		log := logger.With("handler", "ConversionPersistence", "event_type", e.Type())
		log.Info("🟢 [START] Received event", "event", e)

		cde, ok := e.(events.CurrencyConverted)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event type for conversion persistence", "event", e)
			return nil
		}
		err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			repo, err := uow.GetRepository((*transaction.Repository)(nil))
			if err != nil {
				return fmt.Errorf("failed to get transaction repository: %w", err)
			}
			txRepo := repo.(transaction.Repository)

			update := dto.TransactionUpdate{
				OriginalAmount:   &cde.ConversionInfo.OriginalAmount,
				OriginalCurrency: &cde.ConversionInfo.OriginalCurrency,
				ConvertedAmount:  &cde.ConversionInfo.ConvertedAmount,
				ConversionRate:   &cde.ConversionInfo.ConversionRate,
				TargetCurrency:   &cde.ConversionInfo.ConvertedCurrency,
			}

			log.Debug("🔄 [PROCESS] Attempting to update transaction with conversion data", "transaction_id", cde.TransactionID, "update_data", update)
			if err := txRepo.Update(ctx, cde.TransactionID, update); err != nil {
				return fmt.Errorf("failed to persist conversion data for transaction %s: %w", cde.TransactionID, err)
			}
			return nil
		})

		if err != nil {
			log.Error("❌ [ERROR] Failed to execute conversion persistence unit of work", "error", err)
			// Note: Not emitting a failure event here to avoid event loops on infra errors.
			return err
		}

		log.Info("✅ [SUCCESS] Conversion data persisted successfully", "transaction_id", cde.TransactionID)
		return nil
	}
}
