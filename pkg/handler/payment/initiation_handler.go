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
		log.Info("🟢 [START] Received validation event", "event", e)

		// Extract payment details from different validation events
		var userID uuid.UUID
		var accountID uuid.UUID
		var amount int64
		var currency string
		var transactionID uuid.UUID
		var correlationID string

		switch evt := e.(type) {
		case events.DepositBusinessValidatedEvent:
			// Extract from DepositBusinessValidatedEvent
			// evt.UserID and evt.AccountID are string, need to parse
			parsedUserID, err := uuid.Parse(evt.UserID)
			if err != nil {
				log.Error("❌ [ERROR] Failed to parse UserID from DepositBusinessValidatedEvent", "error", err, "user_id", evt.UserID)
				return
			}
			parsedAccountID, err := uuid.Parse(evt.AccountID)
			if err != nil {
				log.Error("❌ [ERROR] Failed to parse AccountID from DepositBusinessValidatedEvent", "error", err, "account_id", evt.AccountID)
				return
			}
			userID = parsedUserID
			accountID = parsedAccountID
			amount = evt.ToAmount.Amount()
			currency = evt.ToAmount.Currency().String()
			transactionID = uuid.New() // Will be set by persistence handler
			correlationID = evt.CorrelationID
			log.Info("🔄 [PROCESS] Processing deposit business validation for payment initiation", 
				"user_id", userID, "account_id", accountID, "amount", amount, "currency", currency, "correlation_id", correlationID)

		case events.WithdrawValidatedEvent:
			// evt.UserID and evt.AccountID are uuid.UUID
			userID = evt.UserID
			accountID = evt.AccountID
			amount = evt.Amount.Amount()
			currency = evt.Amount.Currency().String()
			transactionID = uuid.New() // Will be set by persistence handler
			correlationID = evt.CorrelationID
			log.Info("🔄 [PROCESS] Processing withdraw validation for payment initiation", 
				"user_id", userID, "account_id", accountID, "amount", amount, "currency", currency, "correlation_id", correlationID)

		default:
			log.Warn("⚠️ [WARN] Unexpected event type for payment initiation", "event_type", e.EventType(), "event", e)
			return
		}

		// Initiate payment with the provider
		paymentID, err := paymentProvider.InitiatePayment(ctx, userID, accountID, amount, currency)
		if err != nil {
			log.Error("❌ [ERROR] Payment initiation failed", "error", err, "correlation_id", correlationID)
			return
		}

		log.Info("✅ [SUCCESS] Payment initiated successfully", "payment_id", paymentID, "correlation_id", correlationID)

		// Emit PaymentInitiatedEvent
		paymentEvent := events.PaymentInitiatedEvent{
			PaymentID:     paymentID,
			Status:        "initiated",
			TransactionID: transactionID,
			UserID:        userID,
			CorrelationID: correlationID,
		}

		if err := bus.Publish(ctx, paymentEvent); err != nil {
			log.Error("❌ [ERROR] Failed to publish PaymentInitiatedEvent", "error", err, "correlation_id", correlationID)
			return
		}

		log.Info("📤 [EMIT] PaymentInitiatedEvent published", "event", paymentEvent, "correlation_id", correlationID)
	}
}
