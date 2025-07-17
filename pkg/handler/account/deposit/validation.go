package deposit

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/eventbus"
	commonmapper "github.com/amirasaad/fintech/pkg/handler/account/common"
	"github.com/amirasaad/fintech/pkg/queries"
	"github.com/google/uuid"
)

// DepositValidationHandler handles DepositRequestedEvent, maps DTO to domain, validates, and publishes DepositValidatedEvent.
func DepositValidationHandler(bus eventbus.EventBus, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		de, ok := e.(events.DepositRequestedEvent)
		if !ok {
			logger.Error("DepositValidationHandler: unexpected event type", "event", e)
			return
		}
		getAccountResult := queries.GetAccountResult{
			AccountID: de.AccountID,
			UserID:    de.UserID,
			Balance:   de.Amount,
			Currency:  de.Currency,
		}
		acc, err := commonmapper.MapDTOToAccount(getAccountResult)
		if err != nil {
			logger.Error("DepositValidationHandler: failed to map DTO to domain Account", "error", err, "result", getAccountResult)
			return
		}
		userUUID, err := uuid.Parse(de.UserID)
		if err != nil {
			logger.Error("DepositValidationHandler: invalid userID", "error", err, "userID", de.UserID)
			return
		}
		amount, err := money.New(de.Amount, currency.Code(de.Currency))
		if err != nil {
			return
		}
		if err := acc.ValidateDeposit(userUUID, amount); err != nil {
			logger.Error("DepositValidationHandler: domain validation failed", "error", err)
			return
		}
		_ = bus.Publish(ctx, events.DepositValidatedEvent{
			DepositRequestedEvent: de,
			AccountID:             acc.ID.String(),
		})
	}
}
