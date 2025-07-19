package transfer

import (
	"context"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/google/uuid"
)

// TransferValidationHandler handles TransferRequestedEvent, maps DTO to domain, validates, and publishes TransferValidatedEvent.
func TransferValidationHandler(bus eventbus.EventBus, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With("handler", "TransferValidationHandler", "event_type", e.EventType())
		log.Info("🟢 [START] Received event", "event", e)
		te, ok := e.(events.TransferRequestedEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event type", "event", e)
			return
		}
		if te.SenderUserID == uuid.Nil ||
			te.SourceAccountID == uuid.Nil ||
			te.DestAccountID == uuid.Nil {
			log.Error("❌ [ERROR] Missing or invalid fields", "event", te)
			return
		}
		if te.Amount.AmountFloat() <= 0 {
			log.Error("❌ [ERROR] Amount must be positive", "event", te)
			return
		}
		log.Info("✅ [SUCCESS] Transfer validated, emitting TransferValidatedEvent", "event", te)
		_ = bus.Publish(ctx, events.TransferValidatedEvent{TransferRequestedEvent: te})
	}
}
