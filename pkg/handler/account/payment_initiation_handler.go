package account

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

type PaymentProvider interface {
	InitiatePayment(ctx context.Context, userID, accountID string, amount float64, currency string) (string, error)
}

// PaymentInitiationHandler handles MoneyConvertedEvent, initiates payment, and publishes PaymentInitiatedEvent.
func PaymentInitiationHandler(bus eventbus.EventBus, provider PaymentProvider) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		ce, ok := e.(accountdomain.MoneyConvertedEvent)
		if !ok {
			return
		}
		userID := ce.UserID
		accountID := ce.AccountID
		amount := ce.Amount
		currency := ce.Currency

		paymentID, err := provider.InitiatePayment(ctx, userID, accountID, amount, currency)
		if err != nil {
			slog.Error("PaymentInitiationHandler: provider failed", "error", err)
			return
		}
		_ = bus.Publish(ctx, accountdomain.PaymentInitiatedEvent{
			MoneyConvertedEvent: ce,
			PaymentID:           paymentID,
			Status:              "initiated",
		})
	}
}
