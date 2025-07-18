package transfer

import (
	"context"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

// TransferPersistenceAdapter defines the interface for persisting a transfer operation.
type TransferPersistenceAdapter interface {
	PersistTransfer(ctx context.Context, event events.TransferDomainOpDoneEvent) error
}

// TransferPersistenceHandler handles TransferDomainOpDoneEvent, persists to DB, and publishes TransferPersistedEvent.
func TransferPersistenceHandler(bus eventbus.EventBus, adapter TransferPersistenceAdapter) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		evt, ok := e.(events.TransferDomainOpDoneEvent)
		if !ok {
			return
		}
		if err := adapter.PersistTransfer(ctx, evt); err != nil {
			slog.Error("TransferPersistenceHandler: persistence failed", "error", err)
			return
		}
		_ = bus.Publish(ctx, events.TransferPersistedEvent{TransferDomainOpDoneEvent: evt})
	}
}
