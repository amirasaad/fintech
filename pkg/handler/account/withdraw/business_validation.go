package withdraw

import (
	"context"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/google/uuid"
	"log/slog"
	"time"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

// BusinessValidation performs business validation in account currency after conversion.
// Emits WithdrawValidatedEvent to trigger payment initiation.
func BusinessValidation(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "BusinessValidation", "event_type", e.Type())
		log.Info("🟢 [START] Received event", "event", e)
		wce, ok := e.(events.WithdrawBusinessValidationEvent)
		if !ok {
			log.Debug("🚫 [SKIP] Skipping: unexpected event type in BusinessValidation", "event", e)
			return nil
		}
		log.Info("[DEBUG] Incoming WithdrawBusinessValidationEvent IDs", "user_id", wce.UserID, "account_id", wce.AccountID)
		correlationID := wce.CorrelationID
		if wce.FlowType != "withdraw" {
			log.Debug("🚫 [SKIP] Skipping: not a withdraw flow", "flow_type", wce.FlowType)
			return nil
		}
		accRepoAny, err := uow.GetRepository((*account.Repository)(nil))
		if err != nil {
			log.Error("❌ [ERROR] Failed to get account repository", "error", err)
			return err
		}
		accRepo, ok := accRepoAny.(account.Repository)
		if !ok {
			log.Error("❌ [ERROR] Invalid account repository type", "type", accRepoAny)
			return err
		}

		accRead, err := accRepo.Get(ctx, wce.AccountID)
		if err != nil {
			log.Error("❌ [ERROR] Failed to get account", "error", err, "account_id", wce.AccountID)
			return err
		}
		acc := mapper.MapAccountReadToDomain(accRead)
		// Validate the withdrawal (check sufficient funds, account status, etc.)
		if err := acc.ValidateWithdraw(wce.UserID, wce.Amount); err != nil {
			log.Error("❌ [ERROR] Business validation failed",
				"transaction_id", wce.TransactionID,
				"error", err,
				"user_id", wce.UserID,
				"account_id", wce.AccountID,
				"amount", wce.Amount.String())
			return err
		}
		log.Info("✅ [SUCCESS] Business validation passed after conversion, emitting WithdrawValidatedEvent",
			"user_id", wce.UserID,
			"account_id", wce.AccountID,
			"amount", wce.Amount.Amount(),
			"currency", wce.Amount.Currency().String(),
			"correlation_id", correlationID)

		// Emit PaymentInitiationEvent
		paymentInitiationEvent := events.PaymentInitiationEvent{
			FlowEvent:     wce.FlowEvent,
			ID:            uuid.New(),
			TransactionID: wce.TransactionID,
			Account:       acc,
			Amount:        wce.Amount,
			Timestamp:     time.Now(),
		}
		log.Info("📤 [EMIT] Emitting PaymentInitiationEvent", "event", paymentInitiationEvent, "correlation_id", correlationID.String())
		return bus.Emit(ctx, paymentInitiationEvent)
	}
}
