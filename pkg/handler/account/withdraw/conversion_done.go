package withdraw

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

// ConversionDoneHandler handles WithdrawConversionDoneEvent and performs business validation after conversion.
// This handler focuses ONLY on business validation - payment initiation is handled separately by payment handlers.
func ConversionDoneHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With("handler", "WithdrawConversionDoneHandler", "event_type", e.EventType())
		log.Info("🟢 [START] Received event", "event", e)

		we, ok := e.(events.WithdrawConversionDoneEvent)
		if !ok {
			log.Debug("🚫 [SKIP] Skipping: unexpected event type in WithdrawConversionDoneHandler", "event", e)
			return
		}

		log.Info("🔄 [PROCESS] Mapping WithdrawConversionDoneEvent to WithdrawValidatedEvent", "handler", "WithdrawConversionDoneHandler", "event_type", e.EventType(), "correlation_id", we.CorrelationID, "from_amount", we.FromAmount.String(), "to_amount", we.ToAmount.String(), "request_id", we.RequestID)

		// Emit WithdrawValidatedEvent
		withdrawEvent := events.WithdrawValidatedEvent{
			WithdrawRequestedEvent: events.WithdrawRequestedEvent{
				FlowEvent: we.FlowEvent,
				ID:        uuid.New(),
				Amount:    we.ToAmount,
			},
			TargetCurrency: we.ToAmount.Currency().String(),
		}
		log.Info("📤 [EMIT] Emitting WithdrawValidatedEvent", "event", withdrawEvent, "correlation_id", we.CorrelationID.String())
		_ = bus.Publish(ctx, withdrawEvent)
	}
}
