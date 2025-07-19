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
		logger := slog.Default().With("handler", "TransferDomainOpHandler", "event_type", e.EventType())
		ce, ok := e.(events.TransferConversionDoneEvent)
		if !ok {
			logger.Error("unexpected event type", "event", e)
			return
		}
		// TODO: Load accounts, perform domain transfer, handle errors
		logger.Info("performing domain transfer operation (stub)", "event", ce)

		// Parse UUIDs from strings
		logger.Info("parsing UUIDs from TransferConversionDoneEvent",
			"sourceAccountID", ce.SourceAccountID,
			"targetAccountID", ce.TargetAccountID,
			"senderUserID", ce.SenderUserID)

		sourceAccountID, err := uuid.Parse(ce.SourceAccountID)
		if err != nil {
			logger.Error("failed to parse source account ID", "error", err, "value", ce.SourceAccountID)
			return
		}
		targetAccountID, err := uuid.Parse(ce.TargetAccountID)
		if err != nil {
			logger.Error("failed to parse target account ID", "error", err, "value", ce.TargetAccountID)
			return
		}
		senderUserID, err := uuid.Parse(ce.SenderUserID)
		if err != nil {
			logger.Error("failed to parse sender user ID", "error", err, "value", ce.SenderUserID)
			return
		}

		logger.Info("successfully parsed UUIDs",
			"sourceAccountID", sourceAccountID,
			"targetAccountID", targetAccountID,
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
				EventID:         originalTxID, // Use the original transaction ID
				SourceAccountID: sourceAccountID,
				DestAccountID:   targetAccountID,
				SenderUserID:    senderUserID,
				ReceiverUserID:  senderUserID, // Same user for internal transfer
				Amount:          ce.ToAmount,
				Source:          "transfer",
			}},
			SenderUserID:    senderUserID,
			SourceAccountID: sourceAccountID,
			Amount:          ce.ToAmount,
			Source:          "transfer",
		}

		logger.Info("TransferDomainOpDoneEvent published",
			"event", domainOpEvent,
			"amount", ce.ToAmount,
			"currency", ce.ToAmount.Currency().String())
		_ = bus.Publish(ctx, domainOpEvent)
	}
}
