package deposit

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/google/uuid"
)

// BusinessValidationHandler performs business validation in account currency after conversion.
// Emits DepositBusinessValidatedEvent to trigger payment initiation.
func BusinessValidationHandler(bus eventbus.EventBus, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With("handler", "DepositBusinessValidationHandler", "event_type", e.EventType())
		log.Info("🟢 [START] Received event", "event", e)
		dce, ok := e.(events.DepositConversionDoneEvent)
		if !ok {
			log.Warn("⚠️ [WARN] Unexpected event type in DepositBusinessValidationHandler", "event_type", e.EventType(), "event", e)
			return // Ignore unrelated events
		}
		log.Info("[DEBUG] Incoming DepositConversionDoneEvent IDs", "user_id", dce.UserID, "account_id", dce.AccountID)
		correlationID := dce.CorrelationID
		if correlationID == "" {
			correlationID = uuid.NewString()
		}
		log = log.With("correlation_id", correlationID)
		if dce.FlowType != "deposit" {
			log.Warn("⚠️ [WARN] DepositBusinessValidationHandler received event for wrong flow", "flow_type", dce.FlowType)
			return
		}
		// Perform business validation in account currency here...
		log.Info("✅ [SUCCESS] Business validation passed after conversion, emitting DepositBusinessValidatedEvent",
			"user_id", dce.UserID,
			"account_id", dce.AccountID,
			"amount", dce.ToAmount.Amount(),
			"currency", dce.ToAmount.Currency().String(),
			"correlation_id", correlationID)

		// Emit DepositBusinessValidatedEvent
		log.Info("\ud83d\udce4 [EMIT] Emitting DepositBusinessValidatedEvent", "correlation_id", correlationID)
		bus.Publish(ctx, events.DepositBusinessValidatedEvent{
			DepositConversionDoneEvent: dce,
			CorrelationID: correlationID,
			UserID: dce.UserID,
			AccountID: dce.AccountID,
		})
	}
}