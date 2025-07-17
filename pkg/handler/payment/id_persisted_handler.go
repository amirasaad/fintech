package payment

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
)

// PaymentIdPersistenceHandler handles PaymentInitiatedEvent, updates the transaction with the paymentId, and publishes PaymentIdPersistedEvent.
func PaymentIdPersistenceHandler(bus eventbus.EventBus, uow repository.UnitOfWork) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		slog.Info("PaymentIdPersistenceHandler: received event", "event", e)
		pie, ok := e.(events.PaymentInitiatedEvent)
		if !ok {
			return
		}
		// TODO: Implement actual DB update logic using uow.Do
		_ = bus.Publish(ctx, events.PaymentIdPersistedEvent{
			PaymentInitiatedEvent: pie,
			// Add DB transaction info if needed
		})
	}
}
