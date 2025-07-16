package account

import (
	"context"

	"github.com/amirasaad/fintech/pkg/domain"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

// DepositDomainOpHandler handles PaymentInitiatedEvent, performs the deposit domain operation, and publishes DepositDomainOpDoneEvent.
func DepositDomainOpHandler(bus eventbus.EventBus) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		pe, ok := e.(accountdomain.PaymentInitiatedEvent)
		if !ok {
			// log error or ignore
			return
		}
		// TODO: perform deposit domain operation logic here
		_ = bus.Publish(ctx, accountdomain.DepositDomainOpDoneEvent{PaymentInitiatedEvent: pe})
	}
}
