package account

import (
	"context"

	"github.com/amirasaad/fintech/pkg/domain"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/google/uuid"
)

// TransferValidationHandler handles TransferRequestedEvent, validates, and publishes TransferValidatedEvent.
func TransferValidationHandler(bus eventbus.EventBus) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		te, ok := e.(accountdomain.TransferRequestedEvent)
		if !ok {
			return
		}
		if te.SenderUserID == uuid.Nil || te.SourceAccountID == uuid.Nil || te.DestAccountID == uuid.Nil || te.Amount <= 0 || te.Currency == "" {
			return
		}
		_ = bus.Publish(ctx, accountdomain.TransferValidatedEvent{TransferRequestedEvent: te})
	}
}
