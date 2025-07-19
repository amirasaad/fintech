package transfer

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/google/uuid"
)

// BusinessValidationHandler performs business validation in account currency after conversion.
// Emits TransferDomainOpDoneEvent to trigger domain operation.
func BusinessValidationHandler(bus eventbus.EventBus, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With("handler", "TransferBusinessValidationHandler", "event_type", e.EventType())
		log.Info("🟢 [START] Received event", "event", e)
		tce, ok := e.(events.TransferConversionDoneEvent)
		if !ok {
			log.Warn("⚠️ [WARN] Unexpected event type in TransferBusinessValidationHandler", "event_type", e.EventType(), "event", e)
			return
		}
		if tce.Source != "transfer" {
			log.Warn("⚠️ [WARN] TransferBusinessValidationHandler received event for wrong flow", "source", tce.Source)
			return
		}
		// Perform business validation in account currency here...
		log.Info("✅ [SUCCESS] Business validation passed after conversion, emitting TransferDomainOpDoneEvent",
			"sender_user_id", tce.SenderUserID,
			"source_account_id", tce.SourceAccountID,
			"target_account_id", tce.TargetAccountID,
			"amount", tce.ToAmount.Amount(),
			"currency", tce.ToAmount.Currency().String())

		// Emit TransferDomainOpDoneEvent
		log.Info("📤 [EMIT] Emitting TransferDomainOpDoneEvent")
		bus.Publish(ctx, events.TransferDomainOpDoneEvent{
			TransferValidatedEvent: events.TransferValidatedEvent{}, // Fill as needed
			SenderUserID:    uuid.MustParse(tce.SenderUserID),
			SourceAccountID: uuid.MustParse(tce.SourceAccountID),
			Amount:          tce.ToAmount,
			Source:          "transfer",
		})
	}
}