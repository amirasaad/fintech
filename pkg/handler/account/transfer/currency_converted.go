package transfer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
)

// HandleCurrencyConverted performs domain validation after currency conversion for transfers.
// Emits TransferBusinessValidated event to trigger final persistence.
func HandleCurrencyConverted(
	bus eventbus.Bus,
	uow repository.UnitOfWork,
	logger *slog.Logger,
) func(
	ctx context.Context,
	e events.Event,
) error {
	return func(
		ctx context.Context,
		e events.Event,
	) error {
		log := logger.With(
			"handler", "transfer.HandleCurrencyConverted",
			"event_type", e.Type(),
		)

		log.Info(
			"🟢 [HANDLER] HandleCurrencyConverted received event",
			"event_type", e.Type(),
		)

		// 1. Defensive: Check event type and structure
		tcc, ok := e.(*events.TransferCurrencyConverted)
		if !ok {
			log.Error(
				"❌ [DISCARD] Unexpected event type",
				"event", e,
			)
			return fmt.Errorf("unexpected event type: %T", e)
		}

		log = log.With(
			"user_id", tcc.UserID,
			"account_id", tcc.AccountID,
			"transaction_id", tcc.TransactionID,
			"correlation_id", tcc.CorrelationID,
		)

		// 2. Get account repository
		repoAny, err := uow.GetRepository((*account.Repository)(nil))
		if err != nil {
			log.Error(
				"❌ [ERROR] Failed to get account repository",
				"error", err,
			)
			return err
		}

		accRepo, ok := repoAny.(account.Repository)
		if !ok {
			err = fmt.Errorf("unexpected repository type")
			log.Error(
				"❌ [ERROR] Unexpected repository type",
				"error", err,
			)
			return err
		}

		// Get source account DTO
		sourceAccDto, err := accRepo.Get(ctx, tcc.AccountID)
		if err != nil {
			log.Warn(
				"❌ [BUSINESS] Source account not found",
				"account_id", tcc.AccountID,
				"error", err,
			)
			return bus.Emit(ctx, events.NewTransferFailed(
				tcc.OriginalRequest.(*events.TransferRequested),
				"source account not found: "+err.Error(),
			))
		}

		// Map DTO to domain model
		sourceAcc := mapper.MapAccountReadToDomain(sourceAccDto)

		// Get TransferRequested fields once
		tr, ok := tcc.OriginalRequest.(*events.TransferRequested)
		if !ok {
			log.Error(
				"❌ [DISCARD] Unexpected event type",
				"event", tcc.OriginalRequest,
			)
			return fmt.Errorf("unexpected event type: %T", tcc.OriginalRequest)
		}
		// Perform domain validation
		if err := sourceAcc.ValidateTransfer(
			tcc.UserID,
			tr.DestAccountID,
			sourceAcc,
			tcc.ConvertedAmount,
		); err != nil {
			log.Warn(
				"❌ [BUSINESS] Domain validation failed",
				"reason", err,
			)
			return bus.Emit(ctx, events.NewTransferFailed(
				tr,
				err.Error(),
			))
		}

		// 3. Emit success event
		tbv := events.NewTransferBusinessValidated(
			tcc,
		)
		log.Info(
			"✅ [SUCCESS] Domain validation passed, emitting",
			"event_type", tbv.Type(),
		)

		return bus.Emit(ctx, tbv)
	}
}
