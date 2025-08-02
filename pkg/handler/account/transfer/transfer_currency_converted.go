package transfer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
)

// TransferCurrencyConverted performs domain validation after currency conversion for transfers.
// Emits TransferBusinessValidated event to trigger final persistence.
func TransferCurrencyConverted(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e events.Event) error {
	return func(ctx context.Context, e events.Event) error {
		log := logger.With("handler", "TransferCurrencyConverted", "event_type", e.Type())

		log.Info("🟢 [HANDLER] TransferCurrencyConverted received event", "event_type", e.Type())

		// 1. Defensive: Check event type and structure
		tce, ok := e.(*events.TransferCurrencyConverted)
		if !ok {
			log.Error("❌ [DISCARD] Unexpected event type", "event", e)
			return nil
		}

		log = log.With(
			"user_id", tce.TransferRequested.UserID,
			"account_id", tce.TransferRequested.AccountID,
			"transaction_id", tce.TransferRequested.TransactionID,
			"correlation_id", tce.TransferRequested.CorrelationID,
		)

		// 2. Get account repository
		repoAny, err := uow.GetRepository((*repository.AccountRepository)(nil))
		if err != nil {
			log.Error("❌ [ERROR] Failed to get account repository", "error", err)
			return err
		}

		accRepo, ok := repoAny.(repository.AccountRepository)
		if !ok {
			err = fmt.Errorf("unexpected repository type")
			log.Error("❌ [ERROR] Unexpected repository type", "error", err)
			return err
		}

		// Get source account DTO
		sourceAccDto, err := accRepo.Get(tce.TransferRequested.AccountID)
		if err != nil {
			log.Warn("❌ [BUSINESS] Source account not found", "account_id", tce.TransferRequested.AccountID, "error", err)
			failureEvent := events.NewTransferFailed(
				tce.TransferRequested.FlowEvent,
				"source account not found: "+err.Error(),
			)
			return bus.Emit(ctx, failureEvent)
		}

		// Map domain Account to DTO AccountRead
		accountRead := &dto.AccountRead{
			ID:        sourceAccDto.ID,
			UserID:    sourceAccDto.UserID,
			Balance:   float64(sourceAccDto.Balance.Amount()) / 100, // Convert from cents to dollars
			Currency:  string(sourceAccDto.Balance.Currency()),
			Status:    "active", // Assuming active status for simplicity
			CreatedAt: sourceAccDto.CreatedAt,
		}

		// Map DTO to domain model
		sourceAcc := mapper.MapAccountReadToDomain(accountRead)

		// Get TransferRequested fields once
		tr := tce.TransferRequested
		// Perform domain validation
		if err := sourceAcc.ValidateWithdraw(tr.UserID, tr.Amount); err != nil {
			log.Warn("❌ [BUSINESS] Domain validation failed", "reason", err)
			failureEvent := events.NewTransferFailed(
				tce.TransferRequested.FlowEvent,
				err.Error(),
			)
			return bus.Emit(ctx, failureEvent)
		}

		// 3. Emit success event
		log.Info("✅ [SUCCESS] Domain validation passed, emitting TransferBusinessValidated")
		businessValidatedEvent := events.NewTransferBusinessValidated(
			tce,
		)

		return bus.Emit(ctx, businessValidatedEvent)
	}
}
