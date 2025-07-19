package payment

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/google/uuid"
)

// PaymentInitiationHandler handles validation events and initiates payment with the provider.
// This is a generic handler that can process DepositValidatedEvent, WithdrawValidatedEvent, etc.
func PaymentInitiationHandler(bus eventbus.EventBus, paymentProvider provider.PaymentProvider, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With(
			"handler", "PaymentInitiationHandler",
			"event_type", e.EventType(),
		)
		log.Info("received validation event", "event", e)

		// Extract payment details from different validation events
		var userID uuid.UUID
		var accountID uuid.UUID
		var amount int64
		var currency string
		var transactionID uuid.UUID

		switch evt := e.(type) {
		case events.DepositValidatedEvent:
			userID = evt.UserID
			accountID = evt.AccountID
			amount = evt.Amount.Amount() // Use Amount() for int64 (smallest currency unit)
			currency = evt.Amount.Currency().String()
			transactionID = uuid.New() // Will be set by persistence handler
			log.Info("processing deposit validation for payment initiation", 
				"user_id", userID, "account_id", accountID, "amount", amount, "currency", currency)

		case events.WithdrawValidatedEvent:
			userID = evt.UserID
			accountID = evt.AccountID
			amount = evt.Amount.Amount() // Use Amount() for int64 (smallest currency unit)
			currency = evt.Amount.Currency().String()
			transactionID = uuid.New() // Will be set by persistence handler
			log.Info("processing withdraw validation for payment initiation", 
				"user_id", userID, "account_id", accountID, "amount", amount, "currency", currency)

		default:
			log.Error("unexpected event type for payment initiation", "event", e)
			return
		}

		// Initiate payment with the provider
		paymentID, err := paymentProvider.InitiatePayment(ctx, userID, accountID, amount, currency)
		if err != nil {
			log.Error("payment initiation failed", "error", err)
			return
		}

		log.Info("payment initiated successfully", "payment_id", paymentID)

		// Emit PaymentInitiatedEvent
		paymentEvent := events.PaymentInitiatedEvent{
			PaymentID:     paymentID,
			Status:        "initiated",
			TransactionID: transactionID,
			UserID:        userID,
		}

		if err := bus.Publish(ctx, paymentEvent); err != nil {
			log.Error("failed to publish PaymentInitiatedEvent", "error", err)
			return
		}

		log.Info("PaymentInitiatedEvent published", "event", paymentEvent)
	}
}
