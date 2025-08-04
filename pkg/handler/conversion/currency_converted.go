// Package conversion handles currency conversion events and persistence logic.
package conversion

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
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
			log.Error("[ERROR] unexpected event",
				"event", e,
				"event_type", fmt.Sprintf("%T", e),
			)
			return errors.New("unexpected event type")
		}

		// Validate TransactionID
		if cc.TransactionID == uuid.Nil {
			log.Error("❌ [ERROR] TransactionID is nil in CurrencyConverted event")
			return errors.New("invalid transaction ID")
		}

		log.Info("💾 [PROGRESS] persisting conversion data", "transaction_id", cc.TransactionID)

		// Validate that we have the required data before persisting
		if cc.ConversionInfo == nil {
			log.Error("❌ [ERROR] ConversionInfo is nil, cannot persist conversion data")
			return errors.New("conversion info is nil")
		}

		// Persist conversion result (stubbed for now)
		if err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			transactionRepoAny, err := uow.GetRepository((*transaction.Repository)(nil))
			if err != nil {
				return err
			}
			transactionRepo := transactionRepoAny.(transaction.Repository)
			return transactionRepo.Update(ctx, cc.TransactionID, dto.TransactionUpdate{
				OriginalAmount:   &cc.ConversionInfo.OriginalAmount,
				ConvertedAmount:  &cc.ConversionInfo.ConvertedAmount,
				OriginalCurrency: &cc.ConversionInfo.OriginalCurrency,
				TargetCurrency:   &cc.ConversionInfo.ConvertedCurrency,
				ConversionRate:   &cc.ConversionInfo.ConversionRate,
			})
		}); err != nil {
			log.Error("❌ [ERROR] Failed to persist conversion data", "error", err)
			return err
		}

		log.Info("✅ [SUCCESS] conversion persisted", "transaction_id", cc.TransactionID)
		return nil
	}
}
