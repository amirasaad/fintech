package withdraw

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
)

// WithdrawCurrencyConverted performs domain validation after currency conversion for withdrawals.
// Emits WithdrawBusinessValidated event to trigger payment initiation.
func WithdrawCurrencyConverted(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e common.Event) error {
	return func(ctx context.Context, e common.Event) error {
		log := logger.With("handler", "WithdrawCurrencyConverted", "event_type", e.Type())
		log.Info("🟢 [START] Received event", "event", e)

		wce, ok := e.(*events.WithdrawCurrencyConverted)
		if !ok {
			log.Debug("🚫 [SKIP] Skipping: unexpected event type in WithdrawCurrencyConverted", "event", e)
			return nil
		}

		log = log.With(
			"user_id", wce.WithdrawRequested.UserID,
			"account_id", wce.WithdrawRequested.AccountID,
			"transaction_id", wce.TransactionID,
			"correlation_id", wce.WithdrawRequested.CorrelationID,
		)

		if wce.WithdrawRequested.FlowType != "withdraw" {
			log.Debug("🚫 [SKIP] Skipping: not a withdraw flow", "flow_type", wce.WithdrawRequested.FlowType)
			return nil
		}

		accRepoAny, err := uow.GetRepository((*account.Repository)(nil))
		if err != nil {
			return err
		}
		accRepo, ok := accRepoAny.(account.Repository)
		if !ok {
			return errors.New("invalid account repository type")
		}

		accRead, err := accRepo.Get(ctx, wce.WithdrawRequested.AccountID)
		if err != nil && err != domain.ErrAccountNotFound {
			log.Error("❌ [ERROR] Failed to get account", "error", err, "account_id", wce.WithdrawRequested.AccountID)
			return err
		}

		if accRead == nil {
			log.Error("❌ [ERROR] Account not found", "account_id", wce.WithdrawRequested.AccountID)
			return domain.ErrAccountNotFound
		}

		acc := mapper.MapAccountReadToDomain(accRead)

		// Perform domain validation
		if err := acc.ValidateWithdraw(wce.WithdrawRequested.UserID, wce.ConvertedAmount); err != nil {
			log.Error("❌ [ERROR] Domain validation failed",
				"transaction_id", wce.TransactionID,
				"error", err,
				"user_id", wce.WithdrawRequested.UserID,
				"account_id", wce.WithdrawRequested.AccountID,
				"amount", wce.ConvertedAmount.String())

			failureEvent := events.NewWithdrawFailed(
				&wce.WithdrawRequested,
				err.Error(),
			)
			return bus.Emit(ctx, failureEvent)
		}

		log.Info("✅ [SUCCESS] Domain validation passed, emitting WithdrawBusinessValidated",
			"user_id", wce.WithdrawRequested.UserID,
			"account_id", wce.WithdrawRequested.AccountID,
			"amount", wce.ConvertedAmount.Amount(),
			"currency", wce.ConvertedAmount.Currency().String(),
			"correlation_id", wce.WithdrawRequested.CorrelationID)

		// Emit WithdrawBusinessValidated event
		businessValidatedEvent := events.NewWithdrawBusinessValidated(wce)

		log.Info("📤 [EMIT] Emitting WithdrawBusinessValidated", "event", businessValidatedEvent, "correlation_id", wce.WithdrawRequested.CorrelationID.String())
		return bus.Emit(ctx, businessValidatedEvent)
	}
}
