// Package conversion handles currency conversion events and persistence logic.
package conversion

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/handler/common"
	"github.com/google/uuid"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/repository"
)

// HandleCurrencyConverted persists CurrencyConverted events.
func HandleCurrencyConverted(
	uow repository.UnitOfWork,
	logger *slog.Logger) func(
	context.Context,
	events.Event,
) error {
	return func(ctx context.Context, e events.Event) error {
		log := logger.With(
			"handler", "conversion.HandleCurrencyConverted",
			"event_type", e.Type(),
		)
		log.Info("🟢 [START] Event received event")

		cc, ok := e.(*events.CurrencyConverted)
		if !ok {
			log.Warn("unexpected event",
				"event", e,
				"event_type", fmt.Sprintf("%T", e),
			)
			// return nil to skip processing
			return nil
		}

		// Validate TransactionID
		if cc.TransactionID == uuid.Nil {
			log.Warn("TransactionID is nil in CurrencyConverted event",
				"user_id", cc.UserID,
				"account_id", cc.AccountID,
				"correlation_id", cc.CorrelationID,
			)
			return nil
		}

		log = log.With(
			"user_id", cc.UserID,
			"account_id", cc.AccountID,
			"transaction_id", cc.TransactionID,
			"correlation_id", cc.CorrelationID,
		)

		log.Info(
			"💾 [PROGRESS] persisting conversion data",
			"transaction_id", cc.TransactionID,
		)

		// Validate that we have the required data before persisting
		if cc.ConversionInfo == nil {
			log.Warn("ConversionInfo is nil, cannot persist conversion data")
			return nil
		}

		// Persist conversion result (stubbed for now)
		if err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			transactionRepo, err := common.GetTransactionRepository(uow, log)
			if err != nil {
				return err
			}
			// Create money object for transaction amount
			amount := cc.ConvertedAmount.Amount()
			currency := cc.ConvertedAmount.Currency().String()

			return transactionRepo.Update(ctx, cc.TransactionID, dto.TransactionUpdate{
				Amount:           &amount,
				Currency:         &currency,
				OriginalCurrency: &cc.ConversionInfo.FromCurrency,
				TargetCurrency:   &cc.ConversionInfo.ToCurrency,
				ConversionRate:   &cc.ConversionInfo.Rate,
			})
		}); err != nil {
			log.Error("Failed to persist conversion data",
				"error", err,
				"transaction_id", cc.TransactionID,
				"user_id", cc.UserID,
				"account_id", cc.AccountID,
			)
			return err
		}

		log.Info(
			"✅ [SUCCESS] conversion persisted",
			"transaction_id", cc.TransactionID,
		)
		return nil
		// NOTE: Conversion fee application is deferred to a future enhancement.
		// This will be implemented as part of the fee calculation system.
	}
}
