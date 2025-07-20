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
func ConversionDoneHandler(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "WithdrawConversionDoneHandler", "event_type", e.Type())
		log.Info("🟢 [START] Received event", "event", e)

		we, ok := e.(events.WithdrawConversionDoneEvent)
		if !ok {
			log.Debug("🚫 [SKIP] Skipping: unexpected event type in WithdrawConversionDoneHandler", "event", e)
			return nil
		}

		log.Info("🔄 [PROCESS] Mapping WithdrawConversionDoneEvent to WithdrawValidatedEvent", "handler", "WithdrawConversionDoneHandler", "event_type", e.Type(), "correlation_id", we.CorrelationID, "from_amount", we.FromAmount.String(), "to_amount", we.ToAmount.String(), "request_id", we.RequestID)

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
		return bus.Emit(ctx, withdrawEvent)
	}
}
