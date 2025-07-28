package withdraw

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/google/uuid"
)

// Validation handles WithdrawRequestedEvent, performs initial stateless validation, and publishes WithdrawValidatedEvent.
func Validation(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e common.Event) error {
	return func(ctx context.Context, e common.Event) error {
		log := logger.With("handler", "Validation", "event_type", e.Type())
		log.Info("🟢 [START] Received event", "event", e)

		we, ok := e.(*events.WithdrawRequestedEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event type", "event", e)
			return nil
		}

		if we.AccountID == uuid.Nil || !we.Amount.IsPositive() {
			log.Error("❌ [ERROR] Invalid withdrawal request", "event", we)
			failedEvent := events.NewWithdrawFailedEvent(
				events.FlowEvent{
					FlowType:      "withdraw",
					UserID:        we.UserID,
					AccountID:     we.AccountID,
					CorrelationID: we.CorrelationID,
				},
				"Invalid withdrawal request data",
				events.WithWithdrawFailureReason("Invalid withdrawal request data"),
			)
			if err := bus.Emit(ctx, failedEvent); err != nil {
				log.Error("failed to emit WithdrawFailedEvent", "error", err)
			}
			return nil
		}

		accRepoAny, err := uow.GetRepository((*account.Repository)(nil))

		if err != nil {
			return err
		}
		accRepo, ok := accRepoAny.(account.Repository)
		if !ok {
			return errors.New("failed to get repo")
		}

		accDto, err := accRepo.Get(ctx, we.AccountID)
		if err != nil {
			log.Error("❌ [ERROR] Failed to get account", "error", err)
			failedEvent := events.NewWithdrawFailedEvent(
				events.FlowEvent{
					FlowType:      "withdraw",
					UserID:        we.UserID,
					AccountID:     we.AccountID,
					CorrelationID: we.CorrelationID,
				},
				"Account not found",
				events.WithWithdrawFailureReason("Account not found"),
			)
			return bus.Emit(ctx, failedEvent)
		}

		// Create a new FlowEvent with the correct values
		flowEvent := events.FlowEvent{
			FlowType:      "withdraw",
			UserID:        accDto.UserID,
			AccountID:     accDto.ID,
			CorrelationID: we.CorrelationID,
		}

		// Create the validated event with the flow event and copy relevant fields from the request
		validatedEvent := events.NewWithdrawValidatedEvent(
			flowEvent.UserID,
			flowEvent.AccountID,
			flowEvent.CorrelationID,
			events.WithWithdrawValidatedFlowEvent(flowEvent),
			events.WithWithdrawRequestedEvent(*we),
			events.WithTargetCurrency(accDto.Currency),
		)

		log.Info("✅ [SUCCESS] Withdraw request validated, emitting WithdrawValidatedEvent", "account_id", we.AccountID, "user_id", we.UserID)
		return bus.Emit(ctx, validatedEvent)
	}
}
