package deposit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/provider/payment"
	"github.com/amirasaad/fintech/pkg/repository"
)

func HandleValidated(
	bus eventbus.Bus,
	uow repository.UnitOfWork,
	paymentProvider payment.Payment,
	logger *slog.Logger,
) eventbus.HandlerFunc {
	return func(ctx context.Context, e events.Event) error {
		log := logger.With(
			"handler", "deposit.HandleValidated",
			"event_type", e.Type(),
		)
		log.Info("🟢 [START] Processing DepositValidated event")

		dv, ok := e.(*events.DepositValidated)
		if !ok {
			log.Error(
				"unexpected event type",
				"event_type", fmt.Sprintf("%T", e),
			)
			return errors.New("unexpected event type")
		}

		log = log.With(
			"user_id", dv.UserID,
			"account_id", dv.AccountID,
			"transaction_id", dv.TransactionID,
			"correlation_id", dv.CorrelationID,
		)

		pi := events.NewPaymentInitiated(&dv.FlowEvent, func(pi *events.PaymentInitiated) {
			pi.TransactionID = dv.TransactionID
			pi.Amount = dv.ConvertedAmount
			pi.UserID = dv.UserID
			pi.AccountID = dv.AccountID
			pi.CorrelationID = dv.CorrelationID
		})
		log.Info(
			"📤 [EMIT] Emitting event",
			"event_type", pi.Type(),
		)
		if err := bus.Emit(ctx, pi); err != nil {
			log.Warn(
				"Failed to emit",
				"event_type", pi.Type(),
				"error", err,
			)
			return fmt.Errorf("failed to emit %s: %w", pi.Type(), err)
		}
		log.Info(
			"📤 [EMITTED] event",
			"event_id", pi.ID,
			"event_type", pi.Type(),
			"payment_id", pi.PaymentID,
			"status", pi.Status,
		)

		return nil
	}
}
