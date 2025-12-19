package withdraw

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/account"
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

		wcc, ok := e.(*events.WithdrawCurrencyConverted)
		if !ok {
			log.Debug(
				"🚫 skipping: unexpected event type in WithdrawCurrencyConverted",
				"event", e,
			)
			return nil
		}

		wr, ok := wcc.OriginalRequest.(*events.WithdrawRequested)
		if !ok {
			log.Debug(
				"🚫 skipping: unexpected event type in WithdrawCurrencyConverted",
				"event", e,
			)
			return nil
		}

		log = log.With(
			"user_id", wcc.UserID,
			"account_id", wcc.AccountID,
			"transaction_id", wcc.TransactionID,
			"correlation_id", wcc.CorrelationID,
		)

		if wcc.FlowType != "withdraw" {
			log.Debug(
				"🚫 skipping: not a withdraw flow",
				"flow_type", wcc.FlowType,
			)
			return nil
		}

		accRepo, err := common.GetAccountRepository(uow, log)
		if err != nil {
			return errors.New("invalid account repository type")
		}

		accRead, err := accRepo.Get(ctx, wcc.AccountID)
		if err != nil && !errors.Is(err, account.ErrAccountNotFound) {
			log.Error(
				"failed to get account",
				"error", err,
				"account_id", wcc.AccountID,
			)
			return err
		}

		if accRead == nil {
			log.Error(
				"account not found",
				"account_id", wcc.AccountID,
			)
			return account.ErrAccountNotFound
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
		if err := acc.ValidateWithdraw(wcc.UserID, wcc.ConvertedAmount); err != nil {
			log.Error(
				"domain validation failed",
				"transaction_id", wcc.TransactionID,
				"error", err,
				"user_id", wcc.UserID,
				"account_id", wcc.AccountID,
				"amount", wcc.ConvertedAmount.String(),
			)

			wf := events.NewWithdrawFailed(
				wr,
				err.Error(),
			)
			return bus.Emit(ctx, wf)
		}

		log.Info(
			"✅ [SUCCESS] Domain validation passed, emitting WithdrawBusinessValidated",
			"user_id", wcc.UserID,
			"account_id", wcc.AccountID,
			"amount", wcc.ConvertedAmount.Amount(),
			"currency", wcc.ConvertedAmount.Currency().String(),
			"correlation_id", wcc.CorrelationID,
		)

		// Emit WithdrawBusinessValidated event
		wv := events.NewWithdrawValidated(wcc)

		log.Info(
			"📤 [EMIT] Emitting event",
			"event_type", wv.Type(),
			"correlation_id", wcc.CorrelationID.String(),
		)
		return bus.Emit(ctx, wv)
	}
}
