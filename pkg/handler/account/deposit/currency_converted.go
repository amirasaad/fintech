package deposit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
)

// HandleCurrencyConverted performs domain validation after currency conversion for deposits.
// Emits DepositBusinessValidated event to trigger payment initiation.
func HandleCurrencyConverted(
	bus eventbus.Bus,
	uow repository.UnitOfWork,
	logger *slog.Logger,
) eventbus.HandlerFunc {
	return func(
		ctx context.Context,
		e events.Event,
	) error {
		log := logger.With(
			"handler", "deposit.HandleCurrencyConverted",
			"event_type", e.Type(),
		)
		log.Info("🟢 [START] Processing DepositCurrencyConverted event")

		dcc, ok := e.(*events.DepositCurrencyConverted)
		if !ok {
			log.Warn(
				"🚫 [SKIP] Skipping: unexpected event type",
				"event", e,
			)
			return nil
		}

		log = log.With(
			"user_id", dcc.UserID,
			"account_id", dcc.AccountID,
			"transaction_id", dcc.TransactionID,
			"correlation_id", dcc.CorrelationID,
		)

		accRepoAny, err := uow.GetRepository((*account.Repository)(nil))
		if err != nil {
			log.Error(
				"Failed to get account repository",
				"error", err,
			)
			return err
		}
		accRepo, ok := accRepoAny.(account.Repository)
		if !ok {
			err = errors.New("invalid account repository type")
			log.Error(
				"Invalid account repository type",
				"type", accRepoAny,
				"error", err,
			)
			return err
		}

		// Get account for validation
		accountID := dcc.AccountID
		userID := dcc.UserID

		// Log detailed information about the event for debugging
		log.Debug(
			"[DEBUG] Processing DepositCurrencyConverted event",
			"event_type", fmt.Sprintf("%T", dcc),
			"transaction_id", dcc.TransactionID,
			"user_id", userID,
			"account_id", accountID,
			"converted_amount", dcc.ConvertedAmount,
			"has_original_request", dcc.OriginalRequest != nil,
			"original_request_type", fmt.Sprintf("%T", dcc.OriginalRequest),
		)

		// Check if OriginalRequest is nil
		if dcc.OriginalRequest == nil {
			log.Warn(
				"[SKIP] Original request is missing",
				"event_id", dcc.ID,
			)
			return nil
		}

		// Type assert the OriginalRequest to DepositRequested
		dr, ok := dcc.OriginalRequest.(*events.DepositRequested)
		if !ok {
			log.Warn(
				"[SKIP] Unexpected original request type",
				"original_request_type", fmt.Sprintf("%T", dcc.OriginalRequest),
			)
			return nil
		}

		// Get the account
		accRead, err := accRepo.Get(ctx, accountID)
		if err != nil {
			log.Warn(
				"Failed to get account",
				"error", err,
				"account_id", accountID,
			)
			return fmt.Errorf("failed to get account: %w", err)
		}
		acc, err := mapper.MapAccountReadToDomain(accRead)
		if err != nil {
			log.Error(
				"Failed to map account read to domain",
				"error", err,
			)
			return fmt.Errorf("failed to map account read to domain: %w", err)
		}

		// Perform domain validation
		if err := acc.ValidateDeposit(userID, dcc.ConvertedAmount); err != nil {
			log.Warn(
				"Domain validation failed",
				"error", err,
			)
			// Create the failed event
			failedEvent := events.NewDepositFailed(dr, err.Error())
			_ = bus.Emit(ctx, failedEvent)
			return nil
		}

		dv := events.NewDepositValidated(dcc)
		log.Info(
			"✅ [SUCCESS] Domain validation passed, emitting",
			"event_type", dv.Type(),
		)

		if err := bus.Emit(ctx, dv); err != nil {
			log.Warn(
				"Failed to emit",
				"event_type", dv.Type(),
				"error", err,
			)
			return fmt.Errorf("failed to emit %s: %w", dv.Type(), err)
		}
		pi := events.NewPaymentInitiated(&dcc.FlowEvent, func(pi *events.PaymentInitiated) {
			pi.TransactionID = dcc.TransactionID
			pi.Amount = dcc.ConvertedAmount
			pi.UserID = dcc.UserID
			pi.AccountID = dcc.AccountID
			pi.CorrelationID = dcc.CorrelationID
		})
		log.Info(
			"Emitting",
			"event_type", pi.Type(),
		)
		if err := bus.Emit(ctx, pi); err != nil {
			log.Warn(
				"Failed to emit",
				"event_type", pi.Type(),
				"error", err,
			)
			return fmt.Errorf("failed to emit %s: %w", pi.Type(), err)
		}
		return nil
	}
}
