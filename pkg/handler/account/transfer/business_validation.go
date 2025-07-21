package transfer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/google/uuid"
)

// BusinessValidation performs checks like sufficient funds after currency conversion.
func BusinessValidation(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "BusinessValidation", "event_type", e.Type())

		// 1. Defensive: Check event type and structure
		cde, ok := e.(events.TransferConversionDoneEvent)
		if !ok {
			log.Error("❌ [DISCARD] Unexpected event type", "event", e)
			return nil
		}
		log = log.With("correlation_id", cde.CorrelationID)
		log.Info("🟢 [START] Received event", "event", cde)

		if cde.FlowType != "transfer" || cde.AccountID == uuid.Nil || cde.ToAmount.IsZero() || cde.ToAmount.IsNegative() {
			log.Error("❌ [DISCARD] Invalid or non-transfer event", "event", cde)
			return nil
		}

		// 2. Perform Business Validation
		repoAny, err := uow.GetRepository((*account.Repository)(nil))
		if err != nil {
			log.Error("❌ [ERROR] Failed to get repository", "error", err)
			return err // Return repository/DB errors directly
		}
		accRepo, ok := repoAny.(account.Repository)
		if !ok {
			err := fmt.Errorf("unexpected repository type")
			log.Error("❌ [ERROR]", "error", err)
			return err
		}

		sourceAccDto, err := accRepo.Get(ctx, cde.AccountID)
		if err != nil {
			log.Warn("❌ [BUSINESS] Business validation failed", "reason", "source account not found")
			failureEvent := events.TransferFailedEvent{
				TransferRequestedEvent: cde.TransferRequestedEvent,
				Reason:                 "source account not found",
			}
			return bus.Emit(ctx, failureEvent) // Emit business failure
		}

		sourceAcc := mapper.MapAccountReadToDomain(sourceAccDto)
		if err := sourceAcc.ValidateWithdraw(cde.UserID, cde.ToAmount); err != nil {
			log.Warn("❌ [BUSINESS] Business validation failed", "reason", err)
			failureEvent := events.TransferFailedEvent{
				TransferRequestedEvent: cde.TransferRequestedEvent,
				Reason:                 err.Error(),
			}
			return bus.Emit(ctx, failureEvent) // Emit business failure
		}

		// 3. Emit success event
		log.Info("✅ [SUCCESS] Business validation passed, emitting TransferDomainOpDoneEvent")
		domainOpEvent := events.TransferDomainOpDoneEvent{
			TransferValidatedEvent: cde.TransferValidatedEvent,
		}

		return bus.Emit(ctx, domainOpEvent)
	}
}
