package deposit

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/mapper"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/google/uuid"
)

// ConversionDoneHandler handles DepositConversionDoneEvent and performs business validation after conversion.
// This handler focuses ONLY on business validation - payment initiation is handled separately by payment handlers.
func ConversionDoneHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With("handler", "DepositConversionDoneHandler", "event_type", e.EventType())
		log.Info("🟢 [START] Received event", "event", e)

		dce, ok := e.(events.DepositConversionDoneEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event type", "event", e)
			return
		}

		// Extract data from the conversion done event
		userID := dce.UserID
		accountID := dce.AccountID
		requestID := dce.RequestID
		convertedAmount := dce.ToAmount

		log.Info("🔄 [PROCESS] Processing deposit conversion done",
			"user_id", userID,
			"account_id", accountID,
			"request_id", requestID,
			"converted_amount", convertedAmount.Amount(),
			"converted_currency", convertedAmount.Currency().String())

		// Load account for business validation
		accUUID, err := uuid.Parse(accountID)
		if err != nil {
			log.Error("❌ [ERROR] Invalid account ID", "account_id", accountID, "error", err)
			return
		}

		repoAny, err := uow.GetRepository((*account.Repository)(nil))
		if err != nil {
			log.Error("❌ [ERROR] Failed to get AccountRepository", "error", err)
			return
		}
		repo := repoAny.(account.Repository)
		accDto, err := repo.Get(ctx, accUUID)
		if err != nil {
			log.Error("❌ [ERROR] Account not found", "account_id", accountID, "error", err)
			return
		}

		acc := mapper.MapAccountReadToDomain(accDto)

		// Validate deposit limits in account currency (after conversion)
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			log.Error("❌ [ERROR] Invalid user ID", "user_id", userID, "error", err)
			return
		}

		if err := acc.ValidateDeposit(userUUID, convertedAmount); err != nil {
			log.Error("❌ [ERROR] Deposit validation failed after conversion", "error", err)
			return
		}

		log.Info("✅ [SUCCESS] Deposit validation passed after conversion",
			"user_id", userID,
			"account_id", accountID,
			"amount", convertedAmount.Amount(),
			"currency", convertedAmount.Currency().String())

		// Emit DepositValidatedEvent to trigger payment initiation
		// This follows the principle: validation → payment initiation
		validatedEvent := events.DepositValidatedEvent{
			DepositRequestedEvent: events.DepositRequestedEvent{
				EventID:   uuid.New(),
				AccountID: accUUID,
				UserID:    userUUID,
				Amount:    convertedAmount,
				Source:    "deposit", // Default source for converted deposits
			},
			AccountID: accUUID,
			Account:   acc,
		}

		if err := bus.Publish(ctx, validatedEvent); err != nil {
			log.Error("❌ [ERROR] Failed to publish DepositValidatedEvent", "error", err)
			return
		}

		log.Info("📤 [EMIT] Emitting DepositConversionDoneEvent for business validation")
	}
}
