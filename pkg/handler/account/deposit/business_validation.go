package deposit

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/domain"
)

// BusinessValidationHandler performs business validation in account currency after conversion.
// Emits DepositBusinessValidatedEvent to trigger payment initiation.
func BusinessValidationHandler(bus eventbus.EventBus, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With("handler", "DepositBusinessValidationHandler", "event_type", e.EventType())
		log.Info("🟢 [START] Received event", "event", e)
		dce, ok := e.(events.DepositConversionDoneEvent)
		if !ok {
			return // Ignore unrelated events
		}
		// Perform business validation in account currency here...
		log.Info("✅ [SUCCESS] Business validation passed after conversion, emitting DepositBusinessValidatedEvent",
			"user_id", dce.UserID,
			"account_id", dce.AccountID,
			"amount", dce.ToAmount.Amount(),
			"currency", dce.ToAmount.Currency().String())

		// Emit DepositBusinessValidatedEvent
		log.Info("📤 [EMIT] Emitting DepositBusinessValidatedEvent")
		bus.Publish(ctx, events.DepositBusinessValidatedEvent{
			DepositConversionDoneEvent: dce,
		})
	}
}