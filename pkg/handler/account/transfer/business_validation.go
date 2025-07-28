package transfer

import (
	"context"
	"fmt"

	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
	"log/slog"
)

// BusinessValidation performs checks like sufficient funds after currency conversion.
func BusinessValidation(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e common.Event) error {
	return func(ctx context.Context, e common.Event) error {
		log := logger.With("handler", "BusinessValidation", "event_type", e.Type())

		// Best practice: use pointer events for handlers
		log.Info("🟢 [HANDLER] BusinessValidation received event", "event_type", e.Type(), "event_pointer", fmt.Sprintf("%T", e))

		// 1. Defensive: Check event type and structure
		cde, ok := e.(*events.TransferBusinessValidationEvent)
		if !ok {
			log.Error("❌ [DISCARD] Unexpected event type", "event", e)
			return nil
		}

		// 2. Get account repository
		repoAny, err := uow.GetRepository((*repository.AccountRepository)(nil))
		if err != nil {
			log.Error("❌ [ERROR] Failed to get account repository", "error", err)
			return err
		}

		accRepo, ok := repoAny.(repository.AccountRepository)
		if !ok {
			err = fmt.Errorf("unexpected repository type")
			log.Error(" [ERROR] Unexpected repository type", "error", err)
			return err
		}

		// Get source account DTO
		sourceAccDto, err := accRepo.Get(cde.AccountID)
		if err != nil {
			log.Warn(" [BUSINESS] Source account not found", "account_id", cde.AccountID, "error", err)
			failureEvent := events.NewTransferFailedEvent(
				cde.UserID,
				cde.AccountID,
				cde.CorrelationID,
				"source account not found: "+err.Error(),
				events.WithTransferFailedRequestedEvent(cde.TransferRequestedEvent),
			)
			return bus.Emit(ctx, failureEvent) // Emit business failure
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
		if err := sourceAcc.ValidateWithdraw(cde.UserID, cde.ConvertedAmount); err != nil {
			log.Warn("❌ [BUSINESS] Business validation failed", "reason", err)
			failureEvent := events.NewTransferFailedEvent(
				cde.UserID,
				cde.AccountID,
				cde.CorrelationID,
				err.Error(),
				events.WithTransferFailedRequestedEvent(cde.TransferRequestedEvent),
			)
			return bus.Emit(ctx, failureEvent) // Emit business failure
		}

		// 3. Emit success event
		log.Info("✅ [SUCCESS] Business validation passed, emitting TransferDomainOpDoneEvent")
		domainOpEvent := events.TransferDomainOpDoneEvent{
			TransferValidatedEvent: cde.TransferValidatedEvent,
			ConversionDoneEvent:    cde.ConversionDoneEvent,
			TransactionID:          cde.TransactionID,
		}

		return bus.Emit(ctx, domainOpEvent)
	}
}
