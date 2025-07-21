package withdraw

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/google/uuid"
)

// Validation handles WithdrawRequestedEvent, performs initial stateless validation, and publishes WithdrawValidatedEvent.
func Validation(bus eventbus.Bus, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "Validation", "event_type", e.Type())
		log.Info("🟢 [START] Received event", "event", e)

		we, ok := e.(events.WithdrawRequestedEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event type", "event", e)
			return nil
		}

		if we.AccountID == uuid.Nil || !we.Amount.IsPositive() {
			log.Error("❌ [ERROR] Invalid withdrawal request", "event", we)
			bus.Emit(ctx, events.WithdrawFailedEvent{WithdrawRequestedEvent: we, Reason: "Invalid withdrawal request data"})
			return nil
		}

		correlationID := uuid.New()
		validatedEvent := events.WithdrawValidatedEvent{
			WithdrawRequestedEvent: we,
			TargetCurrency:         we.Amount.Currency().String(),
		}

		log.Info("✅ [SUCCESS] Withdraw request validated, emitting WithdrawValidatedEvent", "account_id", we.AccountID, "user_id", we.UserID, "correlation_id", correlationID.String())
		return bus.Emit(ctx, validatedEvent)
	}
}
