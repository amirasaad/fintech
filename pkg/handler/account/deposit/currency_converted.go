package deposit

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
)

// CurrencyConverted performs domain validation after currency conversion for deposits.
// Emits DepositBusinessValidated event to trigger payment initiation.
func CurrencyConverted(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e events.Event) error {
	return func(ctx context.Context, e events.Event) error {
		log := logger.With("handler", "CurrencyConverted", "event_type", e.Type())

		dce, ok := e.(*events.DepositCurrencyConverted)
		if !ok {
			log.Debug("🚫 [SKIP] Skipping: unexpected event type in DepositCurrencyConverted", "event", e)
			return nil
		}

		log = log.With(
			"user_id", dce.DepositRequested.UserID,
			"account_id", dce.DepositRequested.AccountID,
			"transaction_id", dce.DepositRequested.TransactionID,
			"correlation_id", dce.DepositRequested.CorrelationID,
		)

		accRepoAny, err := uow.GetRepository((*account.Repository)(nil))
		if err != nil {
			log.Error("❌ [ERROR] Failed to get account repository", "error", err)
			return err
		}
		accRepo, ok := accRepoAny.(account.Repository)
		if !ok {
			err = errors.New("invalid account repository type")
			log.Error("❌ [ERROR] Invalid account repository type", "type", accRepoAny, "error", err)
			return err
		}

		// Get account for validation
		accountID := dce.DepositRequested.AccountID
		userID := dce.DepositRequested.UserID

		accRead, err := accRepo.Get(ctx, accountID)
		if err != nil {
			log.Error("❌ [ERROR] Failed to get account", "error", err, "account_id", accountID)
			return err
		}

		acc := mapper.MapAccountReadToDomain(accRead)

		// Perform domain validation
		if err := acc.ValidateDeposit(userID, dce.ConvertedAmount); err != nil {
			log.Error("❌ [ERROR] Domain validation failed", "error", err)
			// Create the failed event
			failedEvent := events.NewDepositFailed(dce.DepositRequested, err.Error())
			_ = bus.Emit(ctx, failedEvent)
			return nil
		}

		log.Info("✅ [SUCCESS] Domain validation passed, emitting DepositBusinessValidated", "transaction_id", dce.DepositRequested.TransactionID)

		// Emit DepositBusinessValidated event
		businessValidatedEvent := events.NewDepositBusinessValidated(dce)
		_ = bus.Emit(ctx, businessValidatedEvent)
		return nil
	}
}
