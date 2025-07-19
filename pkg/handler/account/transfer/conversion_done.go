package transfer

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
)

// ConversionDoneHandler handles ConversionDoneEvent for transfer flows and triggers domain transfer operations.
func ConversionDoneHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		logger := logger.With("handler", "TransferConversionDoneHandler")

		te, ok := e.(events.TransferConversionDoneEvent)
		if !ok {
			logger.Debug("🚫 [SKIP] Skipping: unexpected event type in TransferConversionDoneHandler", "event", e)
			return
		}

		logger.Info("🔄 [PROCESS] Mapping TransferConversionDoneEvent to TransferDomainOpDoneEvent", "handler", "TransferConversionDoneHandler", "event_type", e.Type(), "correlation_id", te.CorrelationID, "from_amount", te.FromAmount.String(), "to_amount", te.ToAmount.String(), "request_id", te.RequestID)

		// Emit TransferDomainOpDoneEvent
		transferEvent := events.TransferDomainOpDoneEvent{
			TransferValidatedEvent: te.TransferValidatedEvent,
			// Add any additional fields as needed
		}
		logger.Info("📤 [EMIT] Emitting TransferDomainOpDoneEvent", "event", transferEvent, "correlation_id", te.CorrelationID.String())
		_ = bus.Publish(ctx, transferEvent)

		// TODO: The following logic references removed fields and needs to be revisited for the new event structure.
		// convertedAmount := te.ToAmount
		// senderUserID := te.SenderUserID
		// sourceAccountID := te.SourceAccountID
		// targetAccountID := te.TargetAccountID
		logger.Info("received TransferConversionDoneEvent", "event", te)

		// Validate that the converted amount is positive
		if te.ToAmount.AmountFloat() <= 0 {
			logger.Error("transfer amount must be positive", "converted_amount", te.ToAmount)
			return
		}

		logger.Info("processing transfer conversion done",
			"sender_user_id", te.UserID,
			"source_account_id", te.AccountID,
			// "target_account_id", te.TargetAccountID,
			"converted_amount", te.ToAmount)

		// Perform business validation after conversion
		repoAny, err := uow.GetRepository((*account.Repository)(nil))
		if err != nil {
			logger.Error("failed to get account repository", "error", err)
			return
		}
		repo := repoAny.(account.Repository)

		sourceAccUUID := te.AccountID

		sourceAccDto, err := repo.Get(ctx, sourceAccUUID)
		if err != nil {
			logger.Error("source account not found", "account_id", te.AccountID, "error", err)
			return
		}

		sourceAcc := mapper.MapAccountReadToDomain(sourceAccDto)

		// Validate sufficient funds in source account currency (after conversion)
		senderUserUUID := te.UserID

		if err := sourceAcc.ValidateWithdraw(senderUserUUID, te.ToAmount); err != nil {
			logger.Error("transfer validation failed after conversion", "error", err)
			return
		}

		logger.Info("transfer validation passed after conversion, performing domain transfer operation",
			"sender_user_id", te.UserID,
			"source_account_id", te.AccountID,
			// :TODD: set dest account "target_account_id", te.TargetAccountID,
			"amount", te.ToAmount.Amount(),
			"currency", te.ToAmount.Currency().String())

		// Perform domain transfer operation
		err = uow.Do(ctx, func(uow repository.UnitOfWork) error {
			// Get repositories
			repoAny, err := uow.GetRepository((*account.Repository)(nil))
			if err != nil {
				return err
			}
			accountRepo := repoAny.(account.Repository)

			// Get target account
			targetAccUUID := transferEvent.DestAccountID

			targetAccDto, err := accountRepo.Get(ctx, targetAccUUID)
			if err != nil {
				return err
			}

			targetAcc := mapper.MapAccountReadToDomain(targetAccDto)

			// Get receiver user ID from target account
			receiverUserUUID := targetAcc.UserID

			// Perform the transfer
			if err := sourceAcc.Transfer(senderUserUUID, receiverUserUUID, targetAcc, te.ToAmount, "Internal"); err != nil {
				return err
			}

			// For now, just log the transfer - persistence will be handled by the persistence handler
			logger.Info("domain transfer operation completed",
				"source_account", sourceAcc.ID,
				"target_account", targetAcc.ID,
				"amount", te.ToAmount)

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
