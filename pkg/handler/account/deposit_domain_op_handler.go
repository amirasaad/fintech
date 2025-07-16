package account

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

type DepositService interface {
	Deposit(ctx context.Context, userID, accountID string, amount float64, currency string) error
}

// DepositDomainOpHandler handles PaymentInitiatedEvent, performs the deposit domain operation, and publishes DepositDomainOpDoneEvent.
func DepositDomainOpHandler(bus eventbus.EventBus, service DepositService) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		pe, ok := e.(accountdomain.PaymentInitiatedEvent)
		if !ok {
			return
		}
		userID := pe.UserID
		accountID := pe.AccountID
		amount := pe.Amount
		currency := pe.Currency

		err := service.Deposit(ctx, userID, accountID, amount, currency)
		if err != nil {
			slog.Error("DepositDomainOpHandler: domain op failed", "error", err)
			return
		}
		_ = bus.Publish(ctx, accountdomain.DepositDomainOpDoneEvent{
			PaymentInitiatedEvent: pe,
			UserID:                userID,
			AccountID:             accountID,
			Amount:                amount,
			Currency:              currency,
			// Add additional fields as needed
		})
	}
}
