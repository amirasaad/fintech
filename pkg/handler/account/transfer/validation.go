package transfer

import (
	"context"

	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	commonmapper "github.com/amirasaad/fintech/pkg/handler/account/common"
	"github.com/amirasaad/fintech/pkg/queries"
	"github.com/google/uuid"
)

// TransferValidationHandler handles TransferRequestedEvent, maps DTO to domain, validates, and publishes TransferValidatedEvent.
func TransferValidationHandler(bus eventbus.EventBus, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		te, ok := e.(events.TransferRequestedEvent)
		if !ok {
			logger.Error("TransferValidationHandler: unexpected event type", "event", e)
			return
		}
		if te.SenderUserID == uuid.Nil ||
			te.SourceAccountID == uuid.Nil ||
			te.DestAccountID == uuid.Nil ||
			te.Amount <= 0 ||
			te.Currency == "" {
			logger.Error("TransferValidationHandler: missing or invalid fields", "event", te)
			return
		}
		getAccountResult := queries.GetAccountResult{
			AccountID: te.SourceAccountID.String(),
			UserID:    te.SenderUserID.String(),
			Balance:   te.Amount,
			Currency:  te.Currency,
		}
		acc, err := commonmapper.MapDTOToAccount(getAccountResult)
		if err != nil {
			logger.Error("TransferValidationHandler: failed to map DTO to domain Account", "error", err, "result", getAccountResult)
			return
		}
		userUUID := te.SenderUserID
		if err := acc.Validate(userUUID); err != nil {
			logger.Error("TransferValidationHandler: domain validation failed", "error", err)
			return
		}
		_ = bus.Publish(ctx, events.TransferValidatedEvent{TransferRequestedEvent: te})
	}
}
