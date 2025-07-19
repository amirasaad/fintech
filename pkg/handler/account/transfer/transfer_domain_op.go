package transfer

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/google/uuid"
)

// TransferDomainOpHandler handles TransferConversionDoneEvent, performs the domain transfer, and emits TransferDomainOpDoneEvent.
func TransferDomainOpHandler(bus eventbus.EventBus, operator interface{}) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		logger := slog.Default().With("handler", "TransferDomainOpHandler", "event_type", e.Type())
		ce, ok := e.(events.TransferConversionDoneEvent)
		if !ok {
			logger.Error("unexpected event type", "event", e)
			return
		}
		// TODO: Load accounts, perform domain transfer, handle errors
		logger.Info("performing domain transfer operation (stub)", "event", ce)

		// Parse UUIDs from strings
		logger.Info("parsing UUIDs from TransferConversionDoneEvent",
			"sourceAccountID", ce.AccountID,
			"senderUserID", ce.UserID)

		sourceAccountID := ce.AccountID
		senderUserID := ce.UserID

		logger.Info("successfully parsed UUIDs",
			"sourceAccountID", sourceAccountID,
			"senderUserID", senderUserID)

		// Parse the original transaction ID from the conversion event
		originalTxID, err := uuid.Parse(ce.RequestID)
		if err != nil {
			logger.Error("failed to parse original transaction ID", "error", err, "request_id", ce.RequestID)
			return
		}

		// Create the domain operation done event
		domainOpEvent := events.TransferDomainOpDoneEvent{
			TransferValidatedEvent: events.TransferValidatedEvent{TransferRequestedEvent: events.TransferRequestedEvent{
				ID:             originalTxID, // Use the original transaction ID
				ReceiverUserID: senderUserID, // Same user for internal transfer
				// TODO: Add Amount and Source fields
			}},
		}

		logger.Info("TransferDomainOpDoneEvent published",
			"event", domainOpEvent,
			"amount", ce.ToAmount,
			"currency", ce.ToAmount.Currency().String())
		_ = bus.Publish(ctx, domainOpEvent)
	}
}
