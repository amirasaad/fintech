package deposit

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"

	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

// BusinessValidation performs business validation in account currency after conversion.
// Emits DepositBusinessValidatedEvent to trigger payment initiation.
func BusinessValidation(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e common.Event) error {
	return func(ctx context.Context, e common.Event) error {
		log := logger.With("handler", "DepositBusinessValidationEvent")
		dce, ok := e.(*events.DepositBusinessValidationEvent)
		if !ok {
			log.Debug(" [SKIP] Skipping: unexpected event type in BusinessValidation", "event", e)
			return nil
		}

		accRepoAny, err := uow.GetRepository((*account.Repository)(nil))
		if err != nil {
			log.Error(" [ERROR] Failed to get account repository", "error", err)
			return err
		}
		accRepo, ok := accRepoAny.(account.Repository)
		if !ok {
			err = errors.New("invalid account repository type")
			log.Error("❌ [ERROR] Invalid account repository type", "type", accRepoAny, "error", err)
			return err
		}

		// Access embedded fields directly to avoid QF1008
		depositRequested := dce.DepositRequestedEvent
		accountID := depositRequested.AccountID
		userID := depositRequested.UserID

		accRead, err := accRepo.Get(ctx, accountID)
		if err != nil {
			log.Error("❌ [ERROR] Failed to get account", "error", err, "account_id", accountID)
			return err
		}
		acc := mapper.MapAccountReadToDomain(accRead)
		if err := acc.ValidateDeposit(userID, dce.Amount); err != nil {
			logger.Error("Failed to validate deposit", "error", err)
			// Create the failed event with the original flow event, reason and transaction ID
			failedEvent := events.NewDepositFailedEvent(
				dce.FlowEvent, // Use the original FlowEvent
				err.Error(),
				events.WithDepositFailedTransactionID(dce.DepositValidatedEvent.TransactionID),
			)
			return bus.Emit(ctx, failedEvent)
		}
		log.Info("✅ [SUCCESS] Business validation passed, emitting PaymentInitiationEvent", "transaction_id", dce.DepositValidatedEvent.TransactionID)

		// Create the payment event with FlowEvent from the upstream event
		paymentEvent := events.NewPaymentInitiationEvent(
			events.WithPaymentAccount(acc),
			events.WithPaymentAmount(dce.Amount),
			events.WithPaymentTransactionID(dce.DepositValidatedEvent.TransactionID),
			events.WithFlowEvent(dce.FlowEvent),
		)

		return bus.Emit(ctx, paymentEvent)
	}
}
