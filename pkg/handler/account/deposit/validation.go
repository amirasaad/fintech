package deposit

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/google/uuid"
)

// contextKey is a custom type for context keys to avoid potential collisions
type contextKey string

// String returns the string representation of the context key
func (c contextKey) String() string {
	return string(c)
}

// Validation validates the deposit request and emits DepositValidatedEvent on success.
func Validation(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e common.Event) error {
	return func(ctx context.Context, e common.Event) error {
		log := logger.With("handler", "deposit.Validation", "event_type", e.Type())
		dr, ok := e.(*events.DepositRequestedEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event type", "event", e)
			return nil
		}
		err := dr.Validate()
		if err != nil {
			log.Error("❌ [ERROR] Deposit validation failed", "error", err)
			return nil
		}
		// Log the currency of the incoming deposit
		log.Info("[CHECK] DepositRequestedEvent amount currency", "currency", dr.Amount.Currency().String())
		// If the deposit currency does not match the account's currency, log a warning
		// (This is not an error, but helps catch misrouted events)
		correlationID := uuid.New()
		// Use a custom type for the context key to avoid potential collisions
		var correlationKey contextKey = "correlationID"
		ctx = context.WithValue(ctx, correlationKey, correlationID)
		log = log.With("correlation_id", correlationID)
		log.Info("🟢 [START] Received event", "event", e)
		repoAny, err := uow.GetRepository((*account.Repository)(nil))
		if err != nil {
			log.Error("❌ [ERROR] Failed to get AccountRepository", "error", err)
			return nil
		}
		repo := repoAny.(account.Repository)
		accDto, err := repo.Get(ctx, dr.AccountID)
		if err != nil || accDto == nil {
			log.Error("❌ [ERROR] Account not found", "account_id", dr.AccountID, "error", err)
			return nil
		}
		if accDto.UserID != dr.UserID {
			log.Error("❌ [ERROR] Account not owned by user", "account_id", dr.AccountID, "user_id", dr.UserID)
			return nil
		}

		validatedEvent := events.NewDepositValidatedEvent(
			dr.UserID, dr.AccountID, dr.CorrelationID,
			events.WithDepositRequestedEvent(*dr),
		)
		validatedEvent.FlowEvent.UserID = dr.UserID
		validatedEvent.FlowEvent.AccountID = dr.AccountID

		log.Info("✅ [SUCCESS] Account validated, emitting DepositValidatedEvent", "account_id", accDto.ID, "user_id", accDto.UserID, "correlation_id", correlationID.String())
		return bus.Emit(ctx, validatedEvent)
	}
}
