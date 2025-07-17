package account

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

type PaymentProvider interface {
	InitiatePayment(ctx context.Context, userID, accountID string, amount int64, currency string) (string, error)
}

// PaymentInitiationHandler handles DepositPersistedEvent, initiates payment, and publishes PaymentInitiatedEvent.
func PaymentInitiationHandler(bus eventbus.EventBus, provider PaymentProvider) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		pe, ok := e.(events.DepositPersistedEvent)
		if !ok {
			return
		}
		userID := pe.UserID
		accountID := pe.AccountID
		amount := pe.Amount // int64, smallest unit
		currency := pe.Currency

		paymentID, err := provider.InitiatePayment(ctx, userID, accountID, amount, currency)
		if err != nil {
			slog.Error("PaymentInitiationHandler: provider failed", "error", err)
			return
		}
		_ = bus.Publish(ctx, events.PaymentInitiatedEvent{
			DepositPersistedEvent: pe,
			PaymentID:             paymentID,
			Status:                "initiated",
		})
	}
}
