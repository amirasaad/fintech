package withdraw

import (
	"context"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/google/uuid"
)

// WithdrawValidationHandler handles WithdrawRequestedEvent, performs validation, and publishes WithdrawValidatedEvent.
func WithdrawValidationHandler(bus eventbus.EventBus, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		we, ok := e.(events.WithdrawRequestedEvent)
		if !ok {
			logger.Error("WithdrawValidationHandler: unexpected event type", "event", e)
			return
		}
		if we.Amount.AmountFloat() <= 0 {
			logger.Error("WithdrawValidationHandler: amount must be positive", "event", we)
			return
		}
		if we.UserID == uuid.Nil || we.AccountID == uuid.Nil {
			logger.Error("WithdrawValidationHandler: missing or invalid fields", "event", we)
			return
		}
		//TODO: add withdraw validation
		_ = bus.Publish(ctx, events.WithdrawValidatedEvent{WithdrawRequestedEvent: we})
	}
}
