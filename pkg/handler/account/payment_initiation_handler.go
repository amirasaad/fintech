package account

import (
	"context"

	"github.com/amirasaad/fintech/pkg/domain"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

// PaymentInitiationHandler handles MoneyConvertedEvent, initiates payment, and publishes PaymentInitiatedEvent.
func PaymentInitiationHandler(bus eventbus.EventBus) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		ce, ok := e.(accountdomain.MoneyConvertedEvent)
		if !ok {
			// log error or ignore
			return
		}
		// TODO: perform payment initiation logic here
		_ = bus.Publish(ctx, accountdomain.PaymentInitiatedEvent{MoneyConvertedEvent: ce})
	}
}
