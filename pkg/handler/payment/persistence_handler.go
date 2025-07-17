package payment

import (
	"context"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
)

// DepositPersistenceHandler handles MoneyConvertedEvent, persists to DB, and publishes DepositPersistedEvent.
func DepositPersistenceHandler(bus eventbus.EventBus, uow repository.UnitOfWork) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		mce, ok := e.(events.MoneyConvertedEvent)
		if !ok {
			return
		}
		// TODO: Implement actual DB persistence logic using uow.Do
		_ = bus.Publish(ctx, events.DepositPersistedEvent{
			MoneyCreatedEvent: mce.MoneyCreatedEvent,
			// Add DB transaction info if needed
		})
	}
}
