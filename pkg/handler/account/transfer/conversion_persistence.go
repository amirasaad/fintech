package transfer

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
)

// ConversionPersistence handles TransferConversionDoneEvent and updates the transaction with conversion data.
func ConversionPersistence(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		logger := logger.With("handler", "ConversionPersistence")
		logger.Info("received event", "event", e)

		var convertedAmount float64
		var originalAmount float64
		var originalCurrency string
		var requestID string

		switch evt := e.(type) {
		case events.TransferConversionDoneEvent:
			convertedAmount = evt.ToAmount.AmountFloat()
			originalAmount = evt.FromAmount.AmountFloat()
			originalCurrency = evt.FromAmount.Currency().String()
			requestID = evt.RequestID
			logger.Info("received TransferConversionDoneEvent", "event", evt)
		default:
			logger.Error("unexpected event type for transfer conversion persistence", "event", e)
			return nil
		}

		// Parse the request ID (which is the transaction ID)
		txID, err := uuid.Parse(requestID)
		if err != nil {
			logger.Error("invalid transaction ID in request", "request_id", requestID, "error", err)
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
			"conversion_rate", conversionRate)

		// Update the transaction with conversion data
		err = uow.Do(ctx, func(uow repository.UnitOfWork) error {
			txRepoAny, err := uow.GetRepository((*transaction.Repository)(nil))
			if err != nil {
				logger.Error("failed to get transaction repository", "error", err)
				return err
			}
			txRepo, ok := txRepoAny.(transaction.Repository)
			if !ok {
				return err
			}

			// Update transaction with conversion data
			update := dto.TransactionUpdate{
				OriginalAmount:   &originalAmount,
				OriginalCurrency: &originalCurrency,
				ConversionRate:   &conversionRate,
			}

			if err := txRepo.Update(ctx, txID, update); err != nil {
				logger.Error("failed to update transaction with conversion data", "error", err)
				return err
			}

			logger.Info("transaction updated with conversion data", "transaction_id", txID)
			return nil
		})

		if err != nil {
			logger.Error("failed to persist conversion data", "error", err)
			return nil
		}

		logger.Info("conversion data persisted successfully", "transaction_id", txID)
		return nil
	}
}
