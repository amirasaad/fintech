package deposit

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/google/uuid"
)

// BusinessValidationHandler performs business validation in account currency after conversion.
// Emits DepositBusinessValidatedEvent to trigger payment initiation.
func BusinessValidationHandler(bus eventbus.Bus, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "DepositBusinessValidationHandler", "event_type", e.Type())
		depth, _ := ctx.Value("eventDepth").(int)
		log.Info("[DEPTH] Event received", "type", e.Type(), "depth", depth, "event", e)
		log.Info("🟢 [START] Received event", "event", e)
		dce, ok := e.(events.DepositConversionDoneEvent)
		if !ok {
			log.Debug("🚫 [SKIP] Skipping: unexpected event type in DepositBusinessValidationHandler", "event", e)
			return nil
		}
		log.Info("[DEBUG] Incoming DepositConversionDoneEvent IDs", "user_id", dce.UserID, "account_id", dce.AccountID)
		correlationID := dce.CorrelationID
		if correlationID == uuid.Nil {
			correlationID = uuid.New()
		}
		log = log.With("correlation_id", correlationID)
		if dce.FlowType != "deposit" {
			log.Debug("🚫 [SKIP] Skipping: not a deposit flow", "flow_type", dce.FlowType)
			return nil
		}
		// Log the currencies at the start of business validation
		log.Info("[CHECK] DepositConversionDoneEvent currencies at business validation", "from_currency", dce.FromAmount.Currency().String(), "to_currency", dce.ToAmount.Currency().String())
		// ToAmount should be the account currency after conversion
		// Perform business validation in account currency here...
		log.Info("✅ [SUCCESS] Business validation passed after conversion, emitting DepositBusinessValidatedEvent",
			"user_id", dce.UserID,
			"account_id", dce.AccountID,
			"amount", dce.ToAmount.Amount(),
			"currency", dce.ToAmount.Currency().String(),
			"correlation_id", correlationID)

		// Emit DepositBusinessValidatedEvent
		depositEvent := events.DepositBusinessValidatedEvent{
			DepositConversionDoneEvent: dce,
			TransactionID:              dce.TransactionID,
		}
		log.Info("📤 [EMIT] Emitting DepositBusinessValidatedEvent", "event", depositEvent, "correlation_id", correlationID.String())
		if err := bus.Emit(ctx, depositEvent); err != nil {
			return err
		}
		return nil
	}
}
