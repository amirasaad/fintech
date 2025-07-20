package transfer

import (
	"context"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/google/uuid"
)

// Validation handles TransferRequestedEvent, maps DTO to domain, validates, and publishes TransferValidatedEvent.
func Validation(bus eventbus.Bus, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "Validation", "event_type", e.Type())
		log.Info("🟢 [START] Received event", "event", e)
		tr, ok := e.(events.TransferRequestedEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event type", "event", e)
			return nil
		}
		if tr.DestAccountID == uuid.Nil ||
			tr.ReceiverUserID == uuid.Nil ||
			tr.UserID == uuid.Nil ||
			tr.AccountID == uuid.Nil {
			log.Error("❌ [ERROR] Missing or invalid fields", "event", tr)
			return nil
		}
		if tr.Amount.AmountFloat() <= 0 {
			log.Error("❌ [ERROR] Amount must be positive", "event", tr)
			return nil
		}
		correlationID := uuid.New()
		validatedEvent := events.TransferValidatedEvent{
			TransferRequestedEvent: tr,
		}
		log.Info("✅ [SUCCESS] Transfer validated, emitting TransferValidatedEvent", "dest_account_id", tr.DestAccountID, "receiver_user_id", tr.ReceiverUserID, "correlation_id", correlationID.String())
		return bus.Emit(ctx, validatedEvent)
	}
}
