package deposit

import (
	"context"
	"errors"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/google/uuid"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

// BusinessValidation performs business validation in account currency after conversion.
// Emits DepositBusinessValidatedEvent to trigger payment initiation.
func BusinessValidation(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "DepositBusinessValidationEvent")
		dce, ok := e.(events.DepositBusinessValidationEvent)
		if !ok {
			log.Debug("🚫 [SKIP] Skipping: unexpected event type in BusinessValidation", "event", e)
			return nil
		}
		accRepoAny, err := uow.GetRepository((*account.Repository)(nil))
		if err != nil {
			log.Error("❌ [ERROR] Failed to get account repository", "error", err)
			return err
		}
		accRepo, ok := accRepoAny.(account.Repository)
		if !ok {
			err := errors.New("invalid account repository type")
			log.Error("❌ [ERROR] Invalid account repository type", "type", accRepoAny, "error", err)
			return err
		}

		accRead, err := accRepo.Get(ctx, dce.AccountID)
		if err != nil {
			log.Error("❌ [ERROR] Failed to get account", "error", err, "account_id", dce.AccountID)
			return err
		}
		acc := mapper.MapAccountReadToDomain(accRead)
		if err := acc.ValidateDeposit(dce.UserID, dce.Amount); err != nil {
			// TODO: notify user
			log.Error("❌ [ERROR] Business validation failed", "transaction_id", dce.TransactionID, "err", err)
			return err
		}
		log.Info("✅ [SUCCESS] Business validation passed, emitting DepositBusinessValidatedEvent", "transaction_id", dce.TransactionID)
		return bus.Emit(ctx, events.PaymentInitiationEvent{
			FlowEvent:     dce.FlowEvent,
			ID:            uuid.New(),
			Account:       acc,
			Amount:        dce.Amount,
			TransactionID: dce.TransactionID,
		})
	}
}
