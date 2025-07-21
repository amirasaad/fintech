package transfer

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/google/uuid"
)

// Validation handles TransferRequestedEvent, performs structural validation, and publishes TransferValidatedEvent.
func Validation(bus eventbus.Bus, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "Validation", "event_type", e.Type())

		// 1. Defensive: Check event type
		tr, ok := e.(events.TransferRequestedEvent)
		if !ok {
			log.Error("❌ [DISCARD] Unexpected event type", "event", e)
			return nil
		}
		log = log.With("correlation_id", tr.CorrelationID)
		log.Info("🟢 [START] Received event", "event", tr)

		// 2. Defensive: Structural validation of all required event data
		if tr.ID == uuid.Nil || tr.AccountID == uuid.Nil || tr.DestAccountID == uuid.Nil || tr.UserID == uuid.Nil || tr.Amount.IsZero() || tr.Amount.IsNegative() || tr.Amount.Currency() == "" {
			log.Error("❌ [DISCARD] Malformed event data: missing or invalid required fields", "event", tr)
			return nil
		}

		// 3. Emit validated event if all checks pass
		validatedEvent := events.TransferValidatedEvent{
			TransferRequestedEvent: tr,
		}

		log.Info("✅ [SUCCESS] Transfer structurally validated, emitting TransferValidatedEvent")
		return bus.Emit(ctx, validatedEvent)
	}
}
