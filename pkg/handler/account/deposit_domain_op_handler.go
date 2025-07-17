package account

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

// Add DepositDomainOperator interface for testability
// DepositDomainOperator defines the Deposit method signature for domain op handler
type DepositDomainOperator interface {
	Deposit(ctx context.Context, userID, accountID string, amount float64, currency string) error
}

// DepositDomainOpHandler handles PaymentInitiatedEvent, performs the deposit domain operation, and publishes DepositDomainOpDoneEvent.
func DepositDomainOpHandler(bus eventbus.EventBus, service DepositDomainOperator) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		pe, ok := e.(events.PaymentInitiatedEvent)
		if !ok {
			return
		}
		userID := pe.UserID
		accountID := pe.AccountID
		amount := pe.Amount
		currency := pe.Currency

		err := service.Deposit(ctx, userID, accountID, float64(amount), currency)
		if err != nil {
			slog.Error("DepositDomainOpHandler: domain op failed", "error", err)
			return
		}
		_ = bus.Publish(ctx, events.DepositPersistedEvent{
			MoneyCreatedEvent: events.MoneyCreatedEvent{
				DepositValidatedEvent: events.DepositValidatedEvent{
					DepositRequestedEvent: events.DepositRequestedEvent{
						AccountID: accountID,
						UserID:    userID,
						Amount:    float64(amount),
						Currency:  currency,
					},
					AccountID: accountID,
				},
				Amount:   amount,
				Currency: currency,
			},
			// Add DB transaction info if needed
		})
	}
}
