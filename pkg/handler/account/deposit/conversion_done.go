package deposit

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

// ConversionDoneHandler handles DepositConversionDoneEvent and performs business validation after conversion.
// This handler focuses ONLY on business validation - payment initiation is handled separately by payment handlers.
func ConversionDoneHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With("handler", "DepositConversionDoneHandler", "event_type", e.EventType())
		log.Info("🟢 [START] Received event", "event", e)

		// Only process ConversionDoneEvent; remove old type assertion and error log.
		cde := e.(events.ConversionDoneEvent)
		correlationID := cde.CorrelationID
		if correlationID == "" {
			correlationID = uuid.NewString()
		}
		log = log.With("correlation_id", correlationID)
		log.Info("🔄 [PROCESS] Mapping ConversionDoneEvent to DepositConversionDoneEvent",
			"from_amount", cde.FromAmount,
			"to_amount", cde.ToAmount,
			"request_id", cde.RequestID)

		// Parse the request ID (which is the transaction ID)
		txID, err := uuid.Parse(cde.RequestID)
		if err != nil {
			log.Error("invalid transaction ID in request", "request_id", cde.RequestID, "error", err)
			return
		}

		// Look up the transaction to get UserID and AccountID
		txRepoAny, err := uow.GetRepository((*repository.TransactionRepository)(nil))
		if err != nil {
			log.Error("failed to get transaction repository", "error", err)
			return
		}
		txRepo, ok := txRepoAny.(repository.TransactionRepository)
		if !ok {
			log.Error("failed to cast to TransactionRepository")
			return
		}
		tx, err := txRepo.Get(ctx, txID)
		if err != nil {
			log.Error("failed to get transaction for conversion done", "transaction_id", txID, "error", err)
			return
		}

		userID := tx.UserID.String()
		accountID := tx.AccountID.String()

		if cde.FlowType == "deposit" {
			depositEvent := events.DepositConversionDoneEvent{
				ConversionDoneEvent: cde,
				UserID: userID,
				AccountID: accountID,
				FlowType: "deposit",
				CorrelationID: correlationID,
			}
			log.Info("📤 [EMIT] Emitting DepositConversionDoneEvent", "event", depositEvent, "correlation_id", correlationID)
			_ = bus.Publish(ctx, depositEvent)
		} else {
			log.Info("Skipping DepositConversionDoneEvent emission: flow type is not deposit", "flow_type", cde.FlowType)
		}
	}
}
