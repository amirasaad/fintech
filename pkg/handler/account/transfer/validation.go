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
		te, ok := e.(events.TransferRequestedEvent)
		if !ok {
			logger.Error("TransferValidationHandler: unexpected event type", "event", e)
			return
		}
		if te.SenderUserID == uuid.Nil ||
			te.SourceAccountID == uuid.Nil ||
			te.DestAccountID == uuid.Nil {
			logger.Error("TransferValidationHandler: missing or invalid fields", "event", te)
			return
		}
		if te.Amount.AmountFloat() <= 0 {
			logger.Error("TransferValidationHandler: amount must be positive", "event", te)
			return
		}
		// TODO; transfer validation logic
		_ = bus.Publish(ctx, events.TransferValidatedEvent{TransferRequestedEvent: te})
	}
}
