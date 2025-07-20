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
func ConversionDoneHandler(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "DepositConversionDoneHandler", "event_type", e.Type())
		depth, _ := ctx.Value("eventDepth").(int)
		log.Info("[DEPTH] Event received", "type", e.Type(), "depth", depth, "event", e)
		log.Info("🟢 [START] Received event", "event", e)

		// Only process ConversionDoneEvent; remove old type assertion and error log.
		de, ok := e.(events.DepositConversionDoneEvent)
		if !ok {
			log.Debug("🚫 [SKIP] Skipping: unexpected event type in DepositConversionDoneHandler", "event", e)
			return nil
		}

		log.Info("🔄 [PROCESS] Mapping DepositConversionDoneEvent to DepositBusinessValidatedEvent", "handler", "DepositConversionDoneHandler", "event_type", e.Type(), "correlation_id", de.CorrelationID, "from_amount", de.FromAmount.String(), "to_amount", de.ToAmount.String(), "request_id", de.RequestID)

		// Emit DepositBusinessValidatedEvent
		depositEvent := events.DepositBusinessValidatedEvent{
			DepositConversionDoneEvent: de,
			TransactionID:              de.TransactionID,
		}
		log.Info("📤 [EMIT] Emitting DepositBusinessValidatedEvent", "event", depositEvent, "correlation_id", de.CorrelationID.String())
		return bus.Emit(ctx, depositEvent)
	}
}
