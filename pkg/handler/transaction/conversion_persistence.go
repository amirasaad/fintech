package transaction

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
	"github.comcom/amirasaad/fintech/pkg/dto"
)

// ConversionPersistence handles persisting conversion details for any transaction.
func ConversionPersistence(uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "ConversionPersistence", "event_type", e.Type())

		var cde events.ConversionDoneEvent
		var transactionID uuid.UUID

		// Type switch to handle different conversion events
		switch event := e.(type) {
		case events.DepositConversionDoneEvent:
			cde = event.ConversionDoneEvent
			transactionID = event.TransactionID
			log.Info("🟢 [START] Received DepositConversionDoneEvent", "correlation_id", cde.CorrelationID)
		case events.WithdrawConversionDoneEvent:
			cde = event.ConversionDoneEvent
			transactionID = cde.TransactionID // From embedded event
			log.Info("🟢 [START] Received WithdrawConversionDoneEvent", "correlation_id", cde.CorrelationID)
		default:
			log.Debug("🚫 [SKIP] Skipping: unexpected event type", "event", e)
			return nil
		}

		err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			repo, err := uow.GetRepository((*transaction.Repository)(nil))
			if err != nil {
				return fmt.Errorf("failed to get transaction repository: %w", err)
			}
			txRepo := repo.(transaction.Repository)

			originalAmount := cde.FromAmount.Amount()
			convertedAmount := cde.ToAmount.Amount()
			targetCurrency := cde.ToAmount.Currency().String()

			update := dto.TransactionUpdate{
				OriginalAmount:   &originalAmount,
				OriginalCurrency: &cde.OriginalCurrency,
				ConvertedAmount:  &convertedAmount,
				ConversionRate:   &cde.ConversionRate,
				TargetCurrency:   &targetCurrency,
			}

			log.Info("🔄 [PROCESS] Updating transaction with conversion data", "transaction_id", transactionID)
			if err := txRepo.Update(ctx, transactionID, update); err != nil {
				return fmt.Errorf("failed to persist conversion data for transaction %s: %w", transactionID, err)
			}
			return nil
		})

		if err != nil {
			log.Error("❌ [ERROR] Failed to execute conversion persistence unit of work", "error", err)
			// Note: Not emitting a failure event here to avoid event loops on infra errors.
			return err
		}

		log.Info("✅ [SUCCESS] Conversion data persisted successfully", "transaction_id", transactionID)
		return nil
	}
}
