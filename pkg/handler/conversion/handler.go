// Package conversion handles currency conversion events and logic.
package conversion

import (
	"context"
	"log/slog"
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/google/uuid"
)

// Handler processes ConversionRequestedEvent, performs conversion, and emits ConversionDoneEvent.
// This handler should only be subscribed to ConversionRequestedEvent, not ConversionDoneEvent.
func Handler(bus eventbus.EventBus, converter money.CurrencyConverter, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With(
			"handler", "ConversionHandler",
			"event_type", e.EventType(),
		)
		log.Info("🟢 [START] Conversion handler triggered", "event_type", e.EventType(), "event", e)

		// Only handle ConversionRequestedEvent
		evt, ok := e.(events.ConversionRequestedEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event type for conversion handler - should only receive ConversionRequestedEvent", "event", e)
			return
		}
		fromAmount := evt.FromAmount
		toCurrency := evt.ToCurrency
		requestID := evt.RequestID

		// Perform currency conversion
		convInfo, err := converter.Convert(fromAmount.AmountFloat(), fromAmount.Currency().String(), toCurrency)
		if err != nil {
			log.Error("❌ [ERROR] Currency conversion failed", "error", err, "from", fromAmount.Currency().String(), "to", toCurrency)
			return
		}

		// Create converted amount
		converted, err := money.New(convInfo.ConvertedAmount, currency.Code(toCurrency))
		if err != nil {
			log.Error("❌ [ERROR] Failed to create converted money", "error", err)
			return
		}

		log.Info("🔄 [PROCESS] Conversion completed successfully",
			"from", fromAmount,
			"to", converted,
			"request_id", requestID)

		// Always emit ConversionDoneEvent (no flowType logic)
		doneEvent := events.ConversionDoneEvent{
			EventID:    uuid.New().String(),
			FromAmount: fromAmount,
			ToAmount:   converted,
			RequestID:  requestID,
			Timestamp:  time.Now(),
		}
		if err := bus.Publish(ctx, doneEvent); err != nil {
			log.Error("❌ [ERROR] Failed to publish ConversionDoneEvent", "error", err)
			return
		}
		log.Info("📤 [EMIT] ConversionDoneEvent published", "event", doneEvent)
	}
}
