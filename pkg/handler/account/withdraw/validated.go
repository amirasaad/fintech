package withdraw

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/handler/common"
	"github.com/amirasaad/fintech/pkg/provider/payment"
	"github.com/amirasaad/fintech/pkg/repository"
)

// HandleValidated handles WithdrawValidated events by initiating a payout.
// It's responsible for starting the external payout process to the user's connected account.
// The function follows these steps:
// 1. Validates the withdrawal request
// 2. Retrieves user's Stripe Connect account
// 3. Prepares and initiates the payout
// 4. Emits appropriate events for the transaction lifecycle
func HandleValidated(
	bus eventbus.Bus,
	uow repository.UnitOfWork,
	paymentProvider payment.Payment,
	logger *slog.Logger,
) eventbus.HandlerFunc {
	return func(ctx context.Context, e events.Event) error {
		log := logger.With(
			"handler", "withdraw.HandleValidated",
			"event_type", e.Type(),
		)
		log.Info("🟢 [START] Processing WithdrawValidated event")

		// Type assert to get the withdraw validated event
		wv, ok := e.(*events.WithdrawValidated)
		if !ok {
			err := fmt.Errorf("expected WithdrawValidated event, got %T", e)
			log.Error("unexpected event type", "error", err)
			return err
		}

		// Get the original withdraw request to access bank details
		req, ok := wv.OriginalRequest.(*events.WithdrawRequested)
		if !ok || req == nil {
			err := errors.New("missing or invalid original withdraw request")
			log.Error("invalid original request", "error", err)
			return fmt.Errorf("invalid withdrawal request: %w", err)
		}

		// Validate the withdrawal amount is positive
		if wv.ConvertedAmount.Amount() <= 0 {
			err := fmt.Errorf("invalid withdrawal amount: %d", wv.ConvertedAmount.Amount())
			log.Error("validation failed", "error", err)
			return err
		}

		log = log.With(
			"user_id", wv.UserID,
			"account_id", wv.AccountID,
			"transaction_id", wv.TransactionID,
			"correlation_id", wv.CorrelationID,
		)

		userRepo, err := common.GetUserRepository(uow, log)
		if err != nil {
			log.Error("Failed to get user repo", "error", err)
			return err
		}
		// Get user details to check for Stripe Connect account
		user, err := userRepo.Get(ctx, req.UserID)
		if err != nil {
			err = fmt.Errorf("failed to get user details: %w", err)
			log.Error("validation failed", "error", err)
			return err
		}

		// Get the user's full name for the payout
		var firstName, lastName string
		if user.Names != "" {
			names := strings.Split(user.Names, " ")
			if len(names) < 2 {
				return fmt.Errorf("user names are required to create a Stripe Connect account")
			}
			firstName = names[0]
			lastName = names[1]
		}

		// Prepare the payout parameters
		description := fmt.Sprintf("Withdrawal from account %s", wv.AccountID)
		metadata := map[string]string{
			"correlation_id":     wv.CorrelationID.String(),
			"flow_type":          "withdraw",
			"stripe_account_id":  user.StripeConnectAccountID,
			"bank_account_last4": lastFourDigits(req.BankAccountNumber),
			"bank_routing":       maskSensitive(req.RoutingNumber, 4),
			"user_id":            wv.UserID.String(),
			"account_id":         wv.AccountID.String(),
			"user_email":         user.Email,
			"user_first_name":    firstName,
			"user_last_name":     lastName,
			"amount":             fmt.Sprintf("%.2f", wv.ConvertedAmount.AmountFloat()),
			"currency":           wv.ConvertedAmount.Currency().String(),
		}

		payoutParams := &payment.InitiatePayoutParams{
			UserID:            wv.UserID,
			AccountID:         wv.AccountID,
			PaymentProviderID: user.StripeConnectAccountID,
			TransactionID:     wv.TransactionID,
			Amount:            wv.ConvertedAmount.Amount(),
			Currency:          strings.ToLower(wv.ConvertedAmount.Currency().String()),
			Description:       description,
			Metadata:          metadata,
			Destination: payment.PayoutDestination{
				Type: payment.PayoutDestinationBankAccount,
				BankAccount: &payment.BankAccountDetails{
					AccountNumber: req.BankAccountNumber,
					RoutingNumber: req.RoutingNumber,
				},
			},
		}

		// Log the payout initiation attempt
		log.Info("Initiating payout",
			"amount", fmt.Sprintf("%.2f", float64(payoutParams.Amount)/100),
			"currency", payoutParams.Currency,
			"destination_type", payoutParams.Destination.Type,
		)

		// Initiate the payout
		payout, err := paymentProvider.InitiatePayout(ctx, payoutParams)
		if err != nil {
			log.Error("Failed to initiate payout", "error", err)

			// Emit WithdrawFailed event with detailed error information
			errMsg := fmt.Sprintf("payout initiation failed: %v", err)
			wf := events.NewWithdrawFailed(
				req,
				errMsg,
				events.WithWithdrawFailureReason(err.Error()),
			)

			if emitErr := bus.Emit(ctx, wf); emitErr != nil {
				log.Error(
					"Failed to emit WithdrawFailed event",
					"error", emitErr,
					"original_error", err,
				)
				// Preserve both errors in the error chain for proper error inspection
				// The emit error is more critical (we couldn't notify about the failure),
				// but we also preserve the original payout error for context
				return errors.Join(
					fmt.Errorf("failed to emit WithdrawFailed event: %w", emitErr),
					fmt.Errorf("original payout error: %w", err),
				)
			}

			// Return a user-friendly error
			return fmt.Errorf("could not process withdrawal: %w", err)
		}
		if err := userRepo.Update(ctx, wv.UserID, &dto.UserUpdate{
			StripeConnectAccountID: &payout.PaymentProviderID,
		}); err != nil {
			log.Error("Failed to update user", "error", err)
			return fmt.Errorf("failed to update user: %w", err)
		}

		log.Info("Payout initiated successfully",
			"payout_id", payout.PayoutID,
			"status", payout.Status,
		)

		// Prepare payment processed event with all required details
		paymentID := payout.PayoutID
		paymentStatus := string(payout.Status)

		// Create a copy of the flow event with updated fields
		flowEvent := wv.FlowEvent
		flowEvent.FlowType = "withdraw"

		// Create a new PaymentProcessed event with the payout details
		// without chaining methods that change the static type
		pp := events.NewPaymentProcessed(&flowEvent, func(pp *events.PaymentProcessed) {
			pp.WithPaymentID(paymentID)
			pp.WithStatus(paymentStatus)
			pp.WithAmount(wv.ConvertedAmount)
			pp.WithTransactionID(wv.TransactionID)
		})

		// Emit the payment processed event
		if err := bus.Emit(ctx, pp); err != nil {
			log.Error("Failed to emit Payment.Processed event",
				"error", err,
				"payment_id", paymentID,
			)
			return fmt.Errorf("failed to emit Payment.Processed event: %w", err)
		}

		log.Info("📤 [EMITTED] event",
			"event_id", pp.ID,
			"event_type", pp.Type(),
			"payment_id", paymentID,
			"status", paymentStatus,
		)

		return nil
	}
}

// lastFourDigits returns the last 4 digits of a bank account number
func lastFourDigits(accountNumber string) string {
	if len(accountNumber) <= 4 {
		return accountNumber
	}
	return accountNumber[len(accountNumber)-4:]
}

// maskSensitive masks all but the last n characters of a string
func maskSensitive(s string, visibleChars int) string {
	if len(s) <= visibleChars {
		return strings.Repeat("*", len(s))
	}
	return strings.Repeat("*", len(s)-visibleChars) + s[len(s)-visibleChars:]
}
