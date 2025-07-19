package withdraw

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

// ConversionDoneHandler handles WithdrawConversionDoneEvent and performs business validation after conversion.
// This handler focuses ONLY on business validation - payment initiation is handled separately by payment handlers.
func ConversionDoneHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With("handler", "WithdrawConversionDoneHandler", "event_type", e.EventType())
		log.Info("🟢 [START] Received event", "event", e)

		// Only process ConversionDoneEvent; remove old type assertion and error log.
		cde := e.(events.ConversionDoneEvent)
		log.Info("🔄 [PROCESS] Mapping ConversionDoneEvent to WithdrawConversionDoneEvent",
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
		tx, err := txRepo.Get(txID)
		if err != nil {
			log.Error("failed to get transaction for conversion done", "transaction_id", txID, "error", err)
			return
		}

		userID := tx.UserID.String()
		accountID := tx.AccountID.String()

		withdrawEvent := events.WithdrawConversionDoneEvent{
			ConversionDoneEvent: cde,
			UserID: userID,
			AccountID: accountID,
			FlowType: "withdraw",
		}
		if withdrawEvent.FlowType == "withdraw" {
			log.Info("🟢 [EMIT] Emitting WithdrawConversionDoneEvent", "event", withdrawEvent)
			_ = bus.Publish(ctx, withdrawEvent)
		} else {
			log.Info("Skipping WithdrawConversionDoneEvent emission: flow type is not withdraw", "flow_type", withdrawEvent.FlowType)
		}
	}
}
