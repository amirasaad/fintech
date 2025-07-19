package deposit

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

// ConversionDoneHandler handles DepositConversionDoneEvent and performs business validation after conversion.
// This handler focuses ONLY on business validation - payment initiation is handled separately by payment handlers.
func ConversionDoneHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With("handler", "DepositConversionDoneHandler", "event_type", e.EventType())
		log.Info("🟢 [START] Received event", "event", e)

		// Only process ConversionDoneEvent; remove old type assertion and error log.
		cde := e.(events.ConversionDoneEvent)
		correlationID := cde.CorrelationID
		if correlationID == "" {
			correlationID = uuid.NewString()
		}
		log = log.With("correlation_id", correlationID)
		log.Info("🔄 [PROCESS] Mapping ConversionDoneEvent to DepositConversionDoneEvent",
			"from_amount", cde.FromAmount,
			"to_amount", cde.ToAmount,
			"request_id", cde.RequestID)
		depositEvent := events.DepositConversionDoneEvent{
			ConversionDoneEvent: cde,
			FlowType: "deposit",
			CorrelationID: correlationID,
		}
		log.Info("📤 [EMIT] Emitting DepositConversionDoneEvent", "event", depositEvent, "correlation_id", correlationID)
		_ = bus.Publish(ctx, depositEvent)
	}
}
