package money

import (
	"context"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

// MoneyCreationHandler handles DepositValidatedEvent, converts float64 amount to int64 (smallest unit), and publishes MoneyCreatedEvent.
func MoneyCreationHandler(bus eventbus.EventBus) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		ve, ok := e.(events.DepositValidatedEvent)
		if !ok {
			return
		}
		if ve.Amount <= 0 {
			return
		}
		m, err := money.New(ve.Amount, currency.Code(ve.Currency))
		if err != nil {
			return
		}
		_ = bus.Publish(ctx, events.MoneyCreatedEvent{
			DepositValidatedEvent: ve,
			Amount:                m.Amount(), // int64, smallest unit
			Currency:              m.Currency().String(),
			TargetCurrency:        ve.Currency,
		})
	}
}
