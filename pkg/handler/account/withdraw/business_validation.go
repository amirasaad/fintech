package withdraw

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

// BusinessValidation performs business validation in account currency after conversion.
// Emits WithdrawValidatedEvent to trigger payment initiation.
func BusinessValidation(bus eventbus.Bus, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "BusinessValidation", "event_type", e.Type())
		log.Info("🟢 [START] Received event", "event", e)
		wce, ok := e.(events.WithdrawConversionDoneEvent)
		if !ok {
			log.Debug("🚫 [SKIP] Skipping: unexpected event type in BusinessValidation", "event", e)
			return nil
		}
		log.Info("[DEBUG] Incoming WithdrawConversionDoneEvent IDs", "user_id", wce.UserID, "account_id", wce.AccountID)
		correlationID := wce.CorrelationID
		if wce.FlowType != "withdraw" {
			log.Debug("🚫 [SKIP] Skipping: not a withdraw flow", "flow_type", wce.FlowType)
			return nil
		}
		log.Info("✅ [SUCCESS] Business validation passed after conversion, emitting WithdrawValidatedEvent",
			"user_id", wce.UserID,
			"account_id", wce.AccountID,
			"amount", wce.ToAmount.Amount(),
			"currency", wce.ToAmount.Currency().String(),
			"correlation_id", correlationID)

		// Emit WithdrawValidatedEvent
		withdrawEvent := events.WithdrawBusinessValidatedEvent{
			WithdrawConversionDoneEvent: wce,
			TransactionID:               wce.TransactionID,
		}
		log.Info("📤 [EMIT] Emitting WithdrawValidatedEvent", "event", withdrawEvent, "correlation_id", correlationID.String())
		if err := bus.Emit(ctx, withdrawEvent); err != nil {
			return err
		}
		return nil
	}
}
