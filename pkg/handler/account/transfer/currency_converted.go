package transfer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/handler/common"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
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
				"unexpected event type",
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
		accRepo, err := common.GetAccountRepository(uow, log)
		if err != nil {
			log.Error(
				"failed to get account repository",
				"error", err,
			)
			return err
		}

		// Get source account DTO
		sourceAccDto, err := accRepo.Get(ctx, tcc.AccountID)
		if err != nil {
			log.Warn(
				"source account not found",
				"account_id", tcc.AccountID,
				"error", err,
			)
			return bus.Emit(ctx, events.NewTransferFailed(
				tcc.OriginalRequest.(*events.TransferRequested),
				"source account not found: "+err.Error(),
			))
		}

		// Map DTO to domain model
		sourceAcc, err := mapper.MapAccountReadToDomain(sourceAccDto)
		if err != nil {
			log.Error(
				"failed to map account read to domain",
				"error", err,
			)
			return fmt.Errorf("failed to map account read to domain: %w", err)
		}

		// Get TransferRequested fields once
		tr, ok := tcc.OriginalRequest.(*events.TransferRequested)
		if !ok {
			log.Error(
				"unexpected event type",
				"event", tcc.OriginalRequest,
			)
			return fmt.Errorf("unexpected event type: %T", tcc.OriginalRequest)
		}
		// Get destination account DTO
		destAccDto, err := accRepo.Get(ctx, tr.DestAccountID)
		if err != nil {
			log.Warn(
				"destination account not found",
				"account_id", tr.DestAccountID,
				"error", err,
			)
			return bus.Emit(ctx, events.NewTransferFailed(
				tr,
				"destination account not found: "+err.Error(),
			))
		}

		// Map DTO to domain model
		destAcc, err := mapper.MapAccountReadToDomain(destAccDto)
		if err != nil {
			log.Error(
				"failed to map destination account read to domain",
				"error", err,
			)
			return fmt.Errorf("failed to map destination account read to domain: %w", err)
		}

		// Perform domain validation
		if err := sourceAcc.ValidateTransfer(
			tcc.UserID,
			// Pass the user ID of the destination account owner for validation
			destAcc.UserID,
			destAcc,
			tcc.ConvertedAmount,
		); err != nil {
			log.Warn(
				"domain validation failed",
				"reason", err,
			)
			return bus.Emit(ctx, events.NewTransferFailed(
				tr,
				err.Error(),
			))
		}

		// 3. Emit success event
		tv := events.NewTransferValidated(
			tcc,
		)
		log.Info(
			"✅ [SUCCESS] Domain validation passed, emitting",
			"event_type", tv.Type(),
		)

		return bus.Emit(ctx, tv)
	}
}
