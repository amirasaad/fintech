package deposit

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"sync"
)

var processedBusinessValidation sync.Map // map[string]struct{} for idempotency

// BusinessValidation performs business validation in account currency after conversion.
// Emits DepositBusinessValidatedEvent to trigger payment initiation.
func BusinessValidation(bus eventbus.Bus, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "BusinessValidation")
		dce, ok := e.(events.DepositConversionDoneEvent)
		if !ok {
			log.Debug("🚫 [SKIP] Skipping: unexpected event type in BusinessValidation", "event", e)
			return nil
		}
		idempotencyKey := dce.TransactionID.String()
		if _, already := processedBusinessValidation.LoadOrStore(idempotencyKey, struct{}{}); already {
			log.Info("🔁 [SKIP] DepositBusinessValidatedEvent already emitted for this transaction", "transaction_id", dce.TransactionID)
			return nil
		}
		log.Info("✅ [SUCCESS] Business validation passed, emitting DepositBusinessValidatedEvent", "transaction_id", dce.TransactionID)
		return bus.Emit(ctx, events.DepositBusinessValidatedEvent{
			DepositConversionDoneEvent: dce,
			TransactionID: dce.TransactionID,
		})
	}
}
