package payment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/provider/payment"
)

var processedPaymentInitiated sync.Map // map[string]struct{} for idempotency

// HandleInitiated handles DepositBusinessValidatedEvent and initiates payment for deposits.
func HandleInitiated(
	bus eventbus.Bus,
	paymentProvider payment.Payment,
	logger *slog.Logger,
) eventbus.HandlerFunc {
	return func(ctx context.Context, e events.Event) error {
		log := logger.With(
			"handler", "payment.HandleInitiated",
			"event_type", e.Type(),
		)
		log.Debug("🔄 Handling PaymentInitiated event",
			"event_type", e.Type(),
			"event", fmt.Sprintf("%+v", e),
		)
		pi, ok := e.(*events.PaymentInitiated)
		if !ok {
			log.Error(
				"unexpected event type",
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

		// Call payment provider
		amount := pi.Amount.Amount()
		currency := pi.Amount.Currency().String()
		payment, err := paymentProvider.InitiatePayment(
			ctx,
			&payment.InitiatePaymentParams{
				UserID:        pi.UserID,
				AccountID:     pi.AccountID,
				Amount:        amount,
				Currency:      currency,
				TransactionID: transactionID,
			},
		)
		if err != nil {
			log.Error(
				"Payment initiation failed",
				"error", err,
			)
			return err
		}
		log.Info(
			"✅ [SUCCESS] Initiated payment",
			"transaction_id", transactionID,
			"payment", payment,
		)
		return nil
	}
}
