package transfer

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
)

// ConversionDoneHandler handles ConversionDoneEvent for transfer flows and triggers domain transfer operations.
func ConversionDoneHandler(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		logger := logger.With("handler", "TransferConversionDoneHandler")

		te, ok := e.(events.TransferConversionDoneEvent)
		if !ok {
			logger.Debug("🚫 [SKIP] Skipping: unexpected event type in TransferConversionDoneHandler", "event", e)
			return nil
		}

		logger.Info("🔄 [PROCESS] Mapping TransferConversionDoneEvent to TransferDomainOpDoneEvent", "handler", "TransferConversionDoneHandler", "event_type", e.Type(), "correlation_id", te.CorrelationID, "from_amount", te.FromAmount.String(), "to_amount", te.ToAmount.String(), "request_id", te.RequestID)

		// Emit TransferDomainOpDoneEvent
		transferEvent := events.TransferDomainOpDoneEvent{
			TransferValidatedEvent: te.TransferValidatedEvent,
			ConversionDoneEvent:    te.ConversionDoneEvent,
		}
		logger.Info("📤 [EMIT] Emitting TransferDomainOpDoneEvent", "event", transferEvent, "correlation_id", te.CorrelationID.String())
		return bus.Emit(ctx, transferEvent)
	}
}
