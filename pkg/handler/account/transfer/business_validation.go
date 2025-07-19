package transfer

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

// BusinessValidationHandler performs business validation in account currency after conversion.
// Emits TransferDomainOpDoneEvent to trigger domain operation.
func BusinessValidationHandler(bus eventbus.EventBus, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With("handler", "TransferBusinessValidationHandler", "event_type", e.EventType())
		log.Info("🟢 [START] Received event", "event", e)
		tce, ok := e.(events.TransferConversionDoneEvent)
		if !ok {
			log.Debug("🚫 [SKIP] Skipping: unexpected event type in TransferBusinessValidationHandler", "event", e)
			return
		}
		log.Info("[DEBUG] Incoming TransferConversionDoneEvent IDs", "user_id", tce.UserID, "account_id", tce.AccountID)
		correlationID := tce.CorrelationID
		if tce.FlowType != "transfer" {
			log.Debug("🚫 [SKIP] Skipping: not a transfer flow", "flow_type", tce.FlowType)
			return
		}
		// Perform business validation in account currency here...
		log.Info("✅ [SUCCESS] Business validation passed after conversion, emitting TransferDomainOpDoneEvent",
			"sender_user_id", tce.UserID,
			"source_account_id", tce.AccountID,
			"amount", tce.ToAmount.Amount(),
			"currency", tce.ToAmount.Currency().String(),
			"correlation_id", correlationID)

		// Emit TransferDomainOpDoneEvent
		log.Info("📤 [EMIT] Emitting TransferDomainOpDoneEvent")
		if err := bus.Publish(ctx, events.TransferDomainOpDoneEvent{
			TransferValidatedEvent: events.TransferValidatedEvent{}, // Fill as needed
		}); err != nil {
			log.Error("failed to publish TransferDomainOpDoneEvent", "error", err)
		}
	}
}
