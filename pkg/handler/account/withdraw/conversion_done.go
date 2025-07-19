package withdraw

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

// ConversionDoneHandler handles WithdrawConversionDoneEvent and performs business validation after conversion.
// This handler focuses ONLY on business validation - payment initiation is handled separately by payment handlers.
func ConversionDoneHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		logger := logger.With(
			"handler", "WithdrawConversionDoneHandler",
			"event_type", e.EventType(),
		)
		logger.Info("received conversion done event", "event", e)

		wce, ok := e.(events.WithdrawConversionDoneEvent)
		if !ok {
			logger.Error("unexpected event type", "event", e)
			return
		}

		// Extract data from the conversion done event
		userID := wce.UserID
		accountID := wce.AccountID
		requestID := wce.RequestID
		convertedAmount := wce.ToAmount

		logger.Info("processing withdraw conversion done",
			"user_id", userID,
			"account_id", accountID,
			"request_id", requestID,
			"converted_amount", convertedAmount.Amount(),
			"converted_currency", convertedAmount.Currency().String())

		// Load account for business validation
		accUUID, err := uuid.Parse(accountID)
		if err != nil {
			logger.Error("invalid account ID", "account_id", accountID, "error", err)
			return
		}

		repoAny, err := uow.GetRepository((*account.Repository)(nil))
		if err != nil {
			logger.Error("failed to get AccountRepository", "error", err)
			return
		}
		repo := repoAny.(account.Repository)
		accDto, err := repo.Get(ctx, accUUID)
		if err != nil {
			logger.Error("account not found", "account_id", accountID, "error", err)
			return
		}

		acc := mapper.MapAccountReadToDomain(accDto)

		// Validate sufficient funds in account currency (after conversion)
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			logger.Error("invalid user ID", "user_id", userID, "error", err)
			return
		}

		if err := acc.ValidateWithdraw(userUUID, convertedAmount); err != nil {
			logger.Error("withdraw validation failed after conversion", "error", err)
			return
		}

		logger.Info("withdraw validation passed after conversion",
			"user_id", userID,
			"account_id", accountID,
			"amount", convertedAmount.Amount(),
			"currency", convertedAmount.Currency().String())

		// Emit WithdrawValidatedEvent to trigger payment initiation
		// This follows the principle: validation → payment initiation
		validatedEvent := events.WithdrawValidatedEvent{
			WithdrawRequestedEvent: events.WithdrawRequestedEvent{
				EventID:               uuid.New(),
				AccountID:             accUUID,
				UserID:                userUUID,
				Amount:                convertedAmount,
				BankAccountNumber:     "", // Will be set by original request
				RoutingNumber:         "", // Will be set by original request
				ExternalWalletAddress: "", // Will be set by original request
			},
			TargetCurrency: convertedAmount.Currency().String(),
			Account:        acc,
		}

		if err := bus.Publish(ctx, validatedEvent); err != nil {
			logger.Error("failed to publish WithdrawValidatedEvent", "error", err)
			return
		}

		logger.Info("WithdrawValidatedEvent published to trigger payment initiation", "event", validatedEvent)
	}
}
