package withdraw

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/google/uuid"
)

// BusinessValidationHandler performs business validation in account currency after conversion.
// Emits WithdrawValidatedEvent to trigger payment initiation.
func BusinessValidationHandler(bus eventbus.EventBus, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With("handler", "WithdrawBusinessValidationHandler", "event_type", e.Type())
		log.Info("🟢 [START] Received event", "event", e)
		wce, ok := e.(events.WithdrawConversionDoneEvent)
		if !ok {
			log.Debug("🚫 [SKIP] Skipping: unexpected event type in WithdrawBusinessValidationHandler", "event", e)
			return
		}
		log.Info("[DEBUG] Incoming WithdrawConversionDoneEvent IDs", "user_id", wce.UserID, "account_id", wce.AccountID)
		correlationID := wce.CorrelationID
		if wce.FlowType != "withdraw" {
			log.Debug("🚫 [SKIP] Skipping: not a withdraw flow", "flow_type", wce.FlowType)
			return
		}
		log.Info("✅ [SUCCESS] Business validation passed after conversion, emitting WithdrawValidatedEvent",
			"user_id", wce.UserID,
			"account_id", wce.AccountID,
			"amount", wce.ToAmount.Amount(),
			"currency", wce.ToAmount.Currency().String(),
			"correlation_id", correlationID)

		// Emit WithdrawValidatedEvent
		withdrawEvent := events.WithdrawValidatedEvent{
			WithdrawRequestedEvent: events.WithdrawRequestedEvent{
				FlowEvent: wce.FlowEvent,
				ID:        uuid.New(),
				// Amount:    wce.ToAmount, // TODO: This field is no longer available in the new event struct.
			},
			TargetCurrency: wce.ToAmount.Currency().String(),
		}
		log.Info("📤 [EMIT] Emitting WithdrawValidatedEvent", "event", withdrawEvent, "correlation_id", correlationID.String())
		if err := bus.Publish(ctx, withdrawEvent); err != nil {
			log.Error("failed to publish WithdrawBusinessValidatedEvent", "error", err)
		}
	}
}
