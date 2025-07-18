package transfer

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/google/uuid"
)

// ConversionDoneHandler handles ConversionDoneEvent for transfer flows and triggers domain transfer operations.
func ConversionDoneHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		logger := logger.With("handler", "TransferConversionDoneHandler")

		// Handle both old and new event types for backward compatibility
		var convertedAmount money.Money
		var senderUserID string
		var sourceAccountID string
		var targetAccountID string

		switch evt := e.(type) {
		case events.TransferConversionDoneEvent:
			convertedAmount = evt.ToAmount
			senderUserID = evt.SenderUserID
			sourceAccountID = evt.SourceAccountID
			targetAccountID = evt.TargetAccountID
			logger.Info("received TransferConversionDoneEvent", "event", evt)
		case events.ConversionDoneEvent:
			convertedAmount = evt.ToAmount
			// Extract user and account info from request ID or context
			logger.Info("received ConversionDoneEvent for transfer", "event", evt)
			return
		case events.ConversionDone:
			if evt.FlowType != "transfer" {
				return // Not a transfer flow
			}
			convertedAmount = evt.ConvertedAmount
			// Extract user and account from original event
			if orig, ok := evt.OriginalEvent.(events.TransferValidatedEvent); ok {
				senderUserID = orig.SenderUserID.String()
				sourceAccountID = orig.SourceAccountID.String()
				targetAccountID = orig.DestAccountID.String()
			} else {
				logger.Error("could not extract user/account from original event", "event", evt.OriginalEvent)
				return
			}
		case events.TransferConversionDone:
			convertedAmount = evt.ConvertedAmount
			senderUserID = evt.SenderUserID.String()
			sourceAccountID = evt.SourceAccountID.String()
			targetAccountID = evt.DestAccountID.String()
		default:
			logger.Error("unexpected event type for transfer conversion done", "event", e)
			return
		}

		// Validate that the converted amount is positive
		if convertedAmount.AmountFloat() <= 0 {
			logger.Error("transfer amount must be positive", "converted_amount", convertedAmount)
			return
		}

		logger.Info("processing transfer conversion done",
			"sender_user_id", senderUserID,
			"source_account_id", sourceAccountID,
			"target_account_id", targetAccountID,
			"converted_amount", convertedAmount)

		// Perform business validation after conversion
		repoAny, err := uow.GetRepository((*account.Repository)(nil))
		if err != nil {
			logger.Error("failed to get account repository", "error", err)
			return
		}
		repo := repoAny.(account.Repository)

		sourceAccUUID, err := uuid.Parse(sourceAccountID)
		if err != nil {
			logger.Error("invalid source account ID", "account_id", sourceAccountID, "error", err)
			return
		}

		sourceAccDto, err := repo.Get(ctx, sourceAccUUID)
		if err != nil {
			logger.Error("source account not found", "account_id", sourceAccountID, "error", err)
			return
		}

		sourceAcc := mapper.MapAccountReadToDomain(sourceAccDto)

		// Validate sufficient funds in source account currency (after conversion)
		senderUserUUID, err := uuid.Parse(senderUserID)
		if err != nil {
			logger.Error("invalid sender user ID", "user_id", senderUserID, "error", err)
			return
		}

		if err := sourceAcc.ValidateWithdraw(senderUserUUID, convertedAmount); err != nil {
			logger.Error("transfer validation failed after conversion", "error", err)
			return
		}

		logger.Info("transfer validation passed after conversion, performing domain transfer operation",
			"sender_user_id", senderUserID,
			"source_account_id", sourceAccountID,
			"target_account_id", targetAccountID,
			"amount", convertedAmount.Amount(),
			"currency", convertedAmount.Currency().String())

		// Perform domain transfer operation
		err = uow.Do(ctx, func(uow repository.UnitOfWork) error {
			// Get repositories
			repoAny, err := uow.GetRepository((*account.Repository)(nil))
			if err != nil {
				return err
			}
			accountRepo := repoAny.(account.Repository)

			// Get target account
			targetAccUUID, err := uuid.Parse(targetAccountID)
			if err != nil {
				return err
			}

			targetAccDto, err := accountRepo.Get(ctx, targetAccUUID)
			if err != nil {
				return err
			}

			targetAcc := mapper.MapAccountReadToDomain(targetAccDto)

			// Get receiver user ID from target account
			receiverUserUUID := targetAcc.UserID

			// Perform the transfer
			if err := sourceAcc.Transfer(senderUserUUID, receiverUserUUID, targetAcc, convertedAmount, "Internal"); err != nil {
				return err
			}

			// For now, just log the transfer - persistence will be handled by the persistence handler
			logger.Info("domain transfer operation completed",
				"source_account", sourceAcc.ID,
				"target_account", targetAcc.ID,
				"amount", convertedAmount)

			return nil
		})

		if err != nil {
			logger.Error("domain transfer operation failed", "error", err)
			return
		}

		logger.Info("domain transfer operation completed successfully")
		// Note: TransferDomainOpDoneEvent will be published by TransferDomainOpHandler
		// which is subscribed to TransferConversionDoneEvent
	}
}
