package deposit

import (
	"context"
	"log/slog"
	"sync"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
)

var processedConversionPersistence sync.Map // map[string]struct{} for idempotency

// ConversionPersistence handles DepositConversionDoneEvent and updates the transaction with conversion data.
func ConversionPersistence(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		logger := logger.With("handler", "ConversionPersistence")
		logger.Info("received event", "event", e)

		var convertedAmount float64
		var originalAmount float64
		var originalCurrency string
		var txID uuid.UUID
		var correlationID uuid.UUID

		var evt events.DepositConversionDoneEvent
		switch v := e.(type) {
		case events.DepositConversionDoneEvent:
			correlationID = v.CorrelationID
			convertedAmount = v.ToAmount.AmountFloat()
			originalAmount = v.FromAmount.AmountFloat()
			originalCurrency = v.FromAmount.Currency().String()
			txID = v.TransactionID
			logger = logger.With("correlation_id", correlationID)
			logger.Info("received DepositConversionDoneEvent", "event", v, "correlation_id", correlationID)
			// assign to evt for use after switch
			evt = v
		default:
			logger.Error("unexpected event type for deposit conversion persistence", "event", e)
			return nil
		}

		// Idempotency check: skip if already processed
		idempotencyKey := txID.String()
		if _, already := processedConversionPersistence.LoadOrStore(idempotencyKey, struct{}{}); already {
			logger.Info("🔁 [SKIP] Conversion already persisted for this transaction", "transaction_id", txID)
			return nil
		}

		// Calculate conversion rate (handle division by zero)
		var conversionRate float64
		if originalAmount > 0 {
			conversionRate = convertedAmount / originalAmount
		} else {
			conversionRate = 1.0 // Default rate for zero amounts
		}

		logger.Info("updating transaction with conversion data",
			"transaction_id", txID,
			"original_amount", originalAmount,
			"original_currency", originalCurrency,
			"converted_amount", convertedAmount,
			"conversion_rate", conversionRate,
			"correlation_id", correlationID)

		// Update the transaction with conversion data
		err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			txRepoAny, err := uow.GetRepository((*transaction.Repository)(nil))
			if err != nil {
				logger.Error("failed to get transaction repository", "error", err)
				return err
			}
			txRepo, ok := txRepoAny.(transaction.Repository)
			if !ok {
				return err
			}

			// Correct assignment: originalCurrency is the source currency (USD), targetCurrency is the destination (e.g., JPY)
			originalAmount := evt.FromAmount.AmountFloat()
			convertedAmount := evt.ToAmount.AmountFloat()
			originalCurrency := evt.FromAmount.Currency().String() // Source currency
			conversionRate := evt.ConversionRate
			logger.Info("[CHECK] Conversion data before update", "original_amount", originalAmount, "converted_amount", convertedAmount, "original_currency", originalCurrency, "conversion_rate", conversionRate)
			update := dto.TransactionUpdate{
				OriginalAmount:   &originalAmount,
				OriginalCurrency: &originalCurrency,
				ConversionRate:   &conversionRate,
				// ... other fields as needed ...
			}
			if err := txRepo.Update(ctx, evt.TransactionID, update); err != nil {
				logger.Error("failed to update transaction with conversion data", "error", err)
				return err
			}

			logger.Info("transaction updated with conversion data", "transaction_id", txID, "correlation_id", correlationID)
			return nil
		})

		if err != nil {
			logger.Error("failed to persist conversion data", "error", err)
			return nil
		}

		logger.Info("conversion data persisted successfully", "transaction_id", txID, "correlation_id", correlationID)

		// Do NOT emit DepositConversionDoneEvent here to avoid cycles
		// End of handler
		return nil
	}
}
