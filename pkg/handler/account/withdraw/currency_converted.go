package withdraw

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/handler/common"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
)

// HandleCurrencyConverted performs domain validation after currency conversion for withdrawals.
// Emits WithdrawBusinessValidated event to trigger payment initiation.
func HandleCurrencyConverted(
	bus eventbus.Bus,
	uow repository.UnitOfWork,
	logger *slog.Logger,
) func(
	ctx context.Context,
	e events.Event,
) error {
	return func(ctx context.Context, e events.Event) error {
		log := logger.With(
			"handler", "withdraw.CurrencyConverted",
			"event_type", e.Type(),
		)
		log.Info("🟢 [START] Received event", "event", e)

		wce, ok := e.(*events.WithdrawCurrencyConverted)
		if !ok {
			log.Debug(
				"🚫 skipping: unexpected event type in WithdrawCurrencyConverted",
				"event", e,
			)
			return nil
		}

		wr, ok := wce.OriginalRequest.(*events.WithdrawRequested)
		if !ok {
			log.Debug(
				"🚫 skipping: unexpected event type in WithdrawCurrencyConverted",
				"event", e,
			)
			return nil
		}

		log = log.With(
			"user_id", wce.UserID,
			"account_id", wce.AccountID,
			"transaction_id", wce.TransactionID,
			"correlation_id", wce.CorrelationID,
		)

		if wce.FlowType != "withdraw" {
			log.Debug(
				"🚫 skipping: not a withdraw flow",
				"flow_type", wce.FlowType,
			)
			return nil
		}

		accRepo, err := common.GetAccountRepository(uow, log)
		if err != nil {
			return errors.New("invalid account repository type")
		}

		accRead, err := accRepo.Get(ctx, wce.AccountID)
		if err != nil && !errors.Is(err, domain.ErrAccountNotFound) {
			log.Error(
				"failed to get account",
				"error", err,
				"account_id", wce.AccountID,
			)
			return err
		}

		if accRead == nil {
			log.Error(
				"account not found",
				"account_id", wce.AccountID,
			)
			return domain.ErrAccountNotFound
		}

		acc, err := mapper.MapAccountReadToDomain(accRead)
		if err != nil {
			log.Error(
				"failed to map account read to domain",
				"error", err,
			)
			return err
		}

		// Perform domain validation
		if err := acc.ValidateWithdraw(wce.UserID, wce.ConvertedAmount); err != nil {
			log.Error(
				"domain validation failed",
				"transaction_id", wce.TransactionID,
				"error", err,
				"user_id", wce.UserID,
				"account_id", wce.AccountID,
				"amount", wce.ConvertedAmount.String(),
			)

			failureEvent := events.NewWithdrawFailed(
				wr,
				err.Error(),
			)
			return bus.Emit(ctx, failureEvent)
		}

		log.Info(
			"✅ [SUCCESS] Domain validation passed, emitting WithdrawBusinessValidated",
			"user_id", wce.UserID,
			"account_id", wce.AccountID,
			"amount", wce.ConvertedAmount.Amount(),
			"currency", wce.ConvertedAmount.Currency().String(),
			"correlation_id", wce.CorrelationID,
		)

		// Emit WithdrawBusinessValidated event
		businessValidatedEvent := events.NewWithdrawValidated(wce)

		log.Info(
			"📤 [EMIT] Emitting WithdrawBusinessValidated",
			"event", businessValidatedEvent,
			"correlation_id", wce.CorrelationID.String(),
		)
		return bus.Emit(ctx, businessValidatedEvent)
	}
}
