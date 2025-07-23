package payment

import (
	"context"
	"log/slog"
	"sync"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/google/uuid"
)

var processedPaymentInitiation sync.Map // map[string]struct{} for idempotency

// Initiation handles DepositBusinessValidatedEvent and initiates payment for deposits.
func Initiation(bus eventbus.Bus, paymentProvider provider.PaymentProvider, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "Initiation")
		evt, ok := e.(events.PaymentInitiationEvent)
		if !ok {
			log.Debug("🚫 [SKIP] Skipping: unexpected event type in Initiation", "event", e)
			return nil
		}
		transactionID := evt.TransactionID
		idempotencyKey := transactionID.String()
		if _, already := processedPaymentInitiation.LoadOrStore(idempotencyKey, struct{}{}); already {
			log.Info("🔁 [SKIP] PaymentInitiatedEvent already emitted for this transaction", "transaction_id", transactionID)
			return nil
		}
		log.Info("✅ [SUCCESS] Initiating payment", "transaction_id", transactionID)
		// Call payment provider
		amount := evt.Amount.Amount()
		currency := evt.Amount.Currency().String()
		paymentID, err := paymentProvider.InitiatePayment(ctx, evt.UserID, evt.AccountID, amount, currency)
		if err != nil {
			log.Error("❌ [ERROR] Payment initiation failed", "error", err)
			return err
		}
		log.Info("📤 [EMIT] Emitting PaymentInitiatedEvent", "transaction_id", transactionID, "payment_id", paymentID)
		return bus.Emit(ctx, events.PaymentInitiatedEvent{
			ID:            uuid.New().String(),
			TransactionID: transactionID,
			PaymentID:     paymentID,
			UserID:        evt.UserID,
			AccountID:     evt.AccountID,
			CorrelationID: evt.CorrelationID,
		})
	}
}
