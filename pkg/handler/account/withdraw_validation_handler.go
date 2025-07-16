package account

import (
	"context"

	"github.com/amirasaad/fintech/pkg/domain"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

// WithdrawValidationHandler handles WithdrawRequestedEvent, performs validation, and publishes WithdrawValidatedEvent.
func WithdrawValidationHandler(bus eventbus.EventBus) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		we, ok := e.(accountdomain.WithdrawRequestedEvent)
		if !ok {
			return
		}
		if we.UserID == "" || we.AccountID == "" || we.Amount <= 0 || we.Currency == "" {
			return
		}
		_ = bus.Publish(ctx, accountdomain.WithdrawValidatedEvent{WithdrawRequestedEvent: we})
	}
}
