package payment

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/provider"
)

// PaymentInitiationHandler handles CurrencyConversionDone, initiates payment, and publishes PaymentInitiatedEvent.
func PaymentInitiationHandler(bus eventbus.EventBus, provider provider.PaymentProvider, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With(
			"handler", "PaymentInitiationHandler",
			"event_type", e.EventType(),
		)
		log.Info("received event", "event", e)
		pe, ok := e.(events.CurrencyConversionDone)
		if !ok {
			log.Error("unexpected event type", "event", e)
			return
		}
		log.Info("processing CurrencyConversionDone",
			"transaction_id", pe.TransactionID,
			"user_id", pe.UserID,
			"account_id", pe.AccountID,
			"converted_amount", pe.ConvertedAmount.Amount(),
			"converted_currency", pe.ConvertedAmount.Currency().String(),
		)

		userID := pe.UserID
		accountID := pe.AccountID
		amount := pe.ConvertedAmount.Amount()
		currency := pe.ConvertedAmount.Currency().String()
		log.Info("initiating payment with provider",
			"user_id", userID, "account_id", accountID, "amount", amount, "currency", currency)

		paymentID, err := provider.InitiatePayment(ctx, userID, accountID, amount, currency)
		if err != nil {
			log.Error("provider failed", "error", err)
			return
		}
		log.Info("payment initiated successfully", "payment_id", paymentID)
		log.Info("publishing PaymentInitiatedEvent",
			"transaction_id", pe.TransactionID, "user_id", userID, "payment_id", paymentID)
		_ = bus.Publish(ctx, events.PaymentInitiatedEvent{
			PaymentID:     paymentID,
			Status:        "initiated",
			TransactionID: pe.TransactionID,
			UserID:        userID,
		})
	}
}
