package deposit

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/google/uuid"
)

// ConversionDoneHandler handles ConversionDoneEvent for deposit flows and triggers payment initiation.
func ConversionDoneHandler(bus eventbus.EventBus, uow repository.UnitOfWork, paymentProvider provider.PaymentProvider, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		logger := logger.With("handler", "DepositConversionDoneHandler")

		// Handle both old and new event types for backward compatibility
		var convertedAmount money.Money
		var userID string
		var accountID string
		var requestID string

		switch evt := e.(type) {
		case events.DepositConversionDoneEvent:
			convertedAmount = evt.ToAmount
			userID = evt.UserID
			accountID = evt.AccountID
			requestID = evt.RequestID
			logger.Info("received DepositConversionDoneEvent", "event", evt)
		case events.ConversionDoneEvent:
			// Extract user and account info from request ID or context
			// For now, we'll need to store this in the request ID or use a different approach
			logger.Info("received ConversionDoneEvent for deposit", "event", evt)
			return
		case events.ConversionDone:
			if evt.FlowType != "deposit" {
				return // Not a deposit flow
			}
			convertedAmount = evt.ConvertedAmount
			// Extract user and account from original event
			if orig, ok := evt.OriginalEvent.(events.DepositValidatedEvent); ok {
				userID = orig.UserID.String()
				accountID = orig.AccountID.String()
			} else {
				logger.Error("could not extract user/account from original event", "event", evt.OriginalEvent)
				return
			}
		case events.DepositConversionDone:
			convertedAmount = evt.ConvertedAmount
			userID = evt.UserID.String()
			accountID = evt.AccountID.String()
		default:
			logger.Error("unexpected event type for deposit conversion done", "event", e)
			return
		}

		logger.Info("processing deposit conversion done",
			"user_id", userID,
			"account_id", accountID,
			"converted_amount", convertedAmount)

		// Perform business validation after conversion
		repoAny, err := uow.GetRepository((*account.Repository)(nil))
		if err != nil {
			logger.Error("failed to get account repository", "error", err)
			return
		}
		repo := repoAny.(account.Repository)

		accUUID, err := uuid.Parse(accountID)
		if err != nil {
			logger.Error("invalid account ID", "account_id", accountID, "error", err)
			return
		}

		accDto, err := repo.Get(ctx, accUUID)
		if err != nil {
			logger.Error("account not found", "account_id", accountID, "error", err)
			return
		}

		acc := mapper.MapAccountReadToDomain(accDto)

		// Validate deposit limits in account currency (after conversion)
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			logger.Error("invalid user ID", "user_id", userID, "error", err)
			return
		}

		if err := acc.ValidateDeposit(userUUID, convertedAmount); err != nil {
			logger.Error("deposit validation failed after conversion", "error", err)
			return
		}

		logger.Info("deposit validation passed after conversion, initiating payment",
			"user_id", userID,
			"account_id", accountID,
			"amount", convertedAmount.Amount(),
			"currency", convertedAmount.Currency().String())

		// Initiate payment with converted amount
		var accountUUID uuid.UUID
		var paymentID string

		accountUUID, err = uuid.Parse(accountID)
		if err != nil {
			logger.Error("invalid account ID for payment", "account_id", accountID, "error", err)
			return
		}

		paymentID, err = paymentProvider.InitiatePayment(ctx, userUUID, accountUUID, convertedAmount.Amount(), convertedAmount.Currency().String())
		if err != nil {
			logger.Error("payment initiation failed", "error", err)
			return
		}

		logger.Info("payment initiated successfully", "payment_id", paymentID)

		// Parse the request ID (which is the transaction ID) for the payment event
		txID, err := uuid.Parse(requestID)
		if err != nil {
			logger.Error("invalid transaction ID for payment event", "request_id", requestID, "error", err)
			return
		}

		// Emit payment initiated event
		paymentEvent := events.PaymentInitiatedEvent{
			PaymentID:     paymentID,
			Status:        "initiated",
			TransactionID: txID, // Use the original transaction ID
			UserID:        userUUID,
		}

		if err := bus.Publish(ctx, paymentEvent); err != nil {
			logger.Error("failed to publish PaymentInitiatedEvent", "error", err)
			return
		}

		logger.Info("PaymentInitiatedEvent published", "event", paymentEvent)
	}
}
