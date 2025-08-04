package payment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/provider"
)

var processedPaymentInitiated sync.Map // map[string]struct{} for idempotency

// HandleInitiated handles DepositBusinessValidatedEvent and initiates payment for deposits.
func HandleInitiated(
	bus eventbus.Bus,
	paymentProvider provider.PaymentProvider,
	logger *slog.Logger,
) eventbus.HandlerFunc {
	return func(ctx context.Context, e events.Event) error {
		log := logger.With(
			"handler", "payment.HandleInitiated",
			"event_type", e.Type(),
		)
		pi, ok := e.(*events.PaymentInitiated)
		if !ok {
			log.Error(
				"🚫 [ERROR] unexpected event type",
				"event_type", fmt.Sprintf("%T", e),
			)
			return errors.New("unexpected event type")
		}
		transactionID := pi.TransactionID
		idempotencyKey := transactionID.String()
		if _, already := processedPaymentInitiated.
			LoadOrStore(
				idempotencyKey,
				struct{}{},
			); already {
			log.Info(
				"🔁 [SKIP] PaymentInitiatedEvent already emitted for this transaction",
				"transaction_id", transactionID,
			)
			return nil
		}
		log.Info(
			"✅ [SUCCESS] Initiating payment",
			"transaction_id", transactionID,
		)
		// Call payment provider
		amount := pi.Amount.Amount()
		currency := pi.Amount.Currency().String()
		paymentID, err := paymentProvider.InitiatePayment(
			ctx,
			pi.UserID,
			pi.AccountID,
			amount,
			currency,
		)
		if err != nil {
			log.Error(
				"❌ [ERROR] Payment initiation failed",
				"error", err,
			)
			return err
		}
		log.Info(
			"📤 [EMIT] Emitting PaymentInitiatedEvent",
			"transaction_id", transactionID,
			"payment_id", paymentID,
		)
		// Create a PaymentInitiated event first
		pp := events.NewPaymentProcessed(
			*pi,
		)

		return bus.Emit(ctx, pp)
	}
}
