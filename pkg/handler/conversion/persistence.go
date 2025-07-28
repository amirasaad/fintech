// Package conversion handles currency conversion events and persistence logic.
package conversion

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/repository/transaction"

	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/repository"
)

// Persistence persists ConversionDoneEvent events.
func Persistence(uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, common.Event) error {
	return func(ctx context.Context, e common.Event) error {
		log := logger.With(
			"handler", "Persistence",
			"event_type", e.Type(),
		)
		log.Info("🟢 [START] Received event", "event", e)

		ce, ok := e.(events.ConversionDoneEvent)
		if !ok {
			log.Error("unexpected event",
				"event", e,
				"concrete_type", fmt.Sprintf("%T", e),
				"expected_type", "ConversionDoneEvent")
			return errors.New("unexpected event")
		}
		log.Info("💾 [PROGRESS] persisting conversion data", "transaction_id", ce.TransactionID)
		// Persist conversion result (stubbed for now)
		if err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			transactionRepoAny, err := uow.GetRepository((*transaction.Repository)(nil))
			if err != nil {
				return err
			}
			transactionRepo := transactionRepoAny.(transaction.Repository)
			return transactionRepo.Update(ctx, ce.TransactionID, dto.TransactionUpdate{
				OriginalAmount:   &ce.ConversionInfo.OriginalAmount,
				ConvertedAmount:  &ce.ConversionInfo.ConvertedAmount,
				OriginalCurrency: &ce.ConversionInfo.OriginalCurrency,
				TargetCurrency:   &ce.ConversionInfo.ConvertedCurrency,
				ConversionRate:   &ce.ConversionInfo.ConversionRate,
			})
		}); err != nil {
			log.Error("❌ [ERROR] Failed to persist conversion data", "error", err)
			return err
		}

		log.Info("✅ [SUCCESS] conversion persisted", "transaction_id", ce.TransactionID)
		return nil
	}
}
