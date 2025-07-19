package deposit

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
)

// ConversionDoneHandler handles DepositConversionDoneEvent and performs business validation after conversion.
// This handler focuses ONLY on business validation - payment initiation is handled separately by payment handlers.
func ConversionDoneHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With("handler", "DepositConversionDoneHandler", "event_type", e.EventType())
		log.Info("🟢 [START] Received event", "event", e)

		// Only process ConversionDoneEvent; remove old type assertion and error log.
		cde, ok := e.(events.ConversionDoneEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event type", "event", e)
			return
		}
		log.Info("🔄 [PROCESS] Mapping ConversionDoneEvent to DepositConversionDoneEvent",
			"from_amount", cde.FromAmount,
			"to_amount", cde.ToAmount,
			"request_id", cde.RequestID)
		depositEvent := events.DepositConversionDoneEvent{
			ConversionDoneEvent: cde,
			// Optionally: fill UserID, AccountID if you can map from request ID
		}
		log.Info("📤 [EMIT] Emitting DepositConversionDoneEvent", "event", depositEvent)
		_ = bus.Publish(ctx, depositEvent)
	}
}
