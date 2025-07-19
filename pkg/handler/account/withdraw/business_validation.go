package withdraw

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/google/uuid"
)

// BusinessValidationHandler performs business validation in account currency after conversion.
// Emits WithdrawValidatedEvent to trigger payment initiation.
func BusinessValidationHandler(bus eventbus.EventBus, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With("handler", "WithdrawBusinessValidationHandler", "event_type", e.EventType())
		log.Info("🟢 [START] Received event", "event", e)
		wce, ok := e.(events.WithdrawConversionDoneEvent)
		if !ok {
			log.Warn("⚠️ [WARN] Unexpected event type in WithdrawBusinessValidationHandler", "event_type", e.EventType(), "event", e)
			return
		}
		// Perform business validation in account currency here...
		log.Info("✅ [SUCCESS] Business validation passed after conversion, emitting WithdrawValidatedEvent",
			"user_id", wce.UserID,
			"account_id", wce.AccountID,
			"amount", wce.ToAmount.Amount(),
			"currency", wce.ToAmount.Currency().String())

		// Emit WithdrawValidatedEvent
		log.Info("📤 [EMIT] Emitting WithdrawValidatedEvent")
		bus.Publish(ctx, events.WithdrawValidatedEvent{
			WithdrawRequestedEvent: events.WithdrawRequestedEvent{
				EventID:   uuid.MustParse(wce.ConversionDoneEvent.EventID),
				AccountID: uuid.MustParse(wce.AccountID),
				UserID:    uuid.MustParse(wce.UserID),
				Amount:    wce.ToAmount,
			},
			TargetCurrency: wce.ToAmount.Currency().String(),
		})
	}
}