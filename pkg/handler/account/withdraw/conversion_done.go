package withdraw

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

// ConversionDoneHandler handles WithdrawConversionDoneEvent and performs business validation after conversion.
func ConversionDoneHandler(bus eventbus.Bus, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "WithdrawConversionDoneHandler", "event_type", e.Type())

		we, ok := e.(events.WithdrawConversionDoneEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event", "event", e)
			return nil
		}

		log.Info("🔄 [PROCESS] Mapping WithdrawConversionDoneEvent to WithdrawBusinessValidatedEvent", "correlation_id", we.CorrelationID)

		// Emit WithdrawBusinessValidatedEvent
		validatedEvent := events.WithdrawBusinessValidatedEvent{
			WithdrawConversionDoneEvent: we,
			TransactionID:               we.TransactionID,
		}

		log.Info("📤 [EMIT] Emitting WithdrawBusinessValidatedEvent", "event", validatedEvent, "correlation_id", we.CorrelationID.String())
		return bus.Emit(ctx, validatedEvent)
	}
}
