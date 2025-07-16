package account

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

// TransferDomainOperator defines the interface for performing a domain transfer operation.
type TransferDomainOperator interface {
	Transfer(ctx context.Context, senderUserID, receiverUserID, sourceAccountID, destAccountID string, amount float64, currency string) error
}

// TransferDomainOpHandler handles TransferValidatedEvent, performs the domain transfer, and publishes TransferDomainOpDoneEvent.
func TransferDomainOpHandler(bus eventbus.EventBus, operator TransferDomainOperator) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		te, ok := e.(accountdomain.TransferValidatedEvent)
		if !ok {
			return
		}
		err := operator.Transfer(
			ctx,
			te.SenderUserID.String(),
			te.ReceiverUserID.String(),
			te.SourceAccountID.String(),
			te.DestAccountID.String(),
			te.Amount,
			te.Currency,
		)
		if err != nil {
			slog.Error("TransferDomainOpHandler: domain op failed", "error", err)
			return
		}
		_ = bus.Publish(ctx, accountdomain.TransferDomainOpDoneEvent{TransferValidatedEvent: te})
	}
}
