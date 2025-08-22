// Package conversion handles currency conversion events and logic.
package conversion

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	exchangeprovider "github.com/amirasaad/fintech/pkg/provider/exchange"
	"github.com/amirasaad/fintech/pkg/registry"
	"github.com/amirasaad/fintech/pkg/service/exchange"
)

// HandleRequested processes ConversionRequestedEvent and
// delegates to a flow-specific factory to create the next event.
func HandleRequested(
	bus eventbus.Bus,
	exchangeRegistry registry.Provider,
	exchangeRateProvider exchangeprovider.Exchange,
	logger *slog.Logger,
	factories map[string]EventFactory,
) func(ctx context.Context, e events.Event) error {
	return func(ctx context.Context, e events.Event) error {
		log := logger.With(
			"handler", "Conversion.HandleRequested",
			"event_type", e.Type(),
		)
		log.Info("🟢 [START] Event received")

		ccr, ok := e.(*events.CurrencyConversionRequested)
		if !ok {
			log.Error(
				"Unexpected event type",
				"event", e,
			)
			return fmt.Errorf("unexpected event type %T", e)
		}

		log.Debug(
			"ConversionRequestedEvent details",
			"event", ccr,
		)
		// Use the factory map to get the correct event factory for the flow type.
		factory, found := factories[ccr.FlowType]
		if !found {
			log.Error(
				"Unknown flow type in ConversionRequestedEvent, discarding",
				"flow_type", ccr.FlowType,
			)
			return fmt.Errorf("unknown flow type %s", ccr.FlowType)
		}

		srv := exchange.New(exchangeRegistry, exchangeRateProvider, log)

		convertedMoney,
			convInfo,
			err := srv.
			Convert(
				ctx,
				ccr.Amount,
				ccr.To,
			)
		if err != nil {
			log.Error(
				"Failed to convert currency",
				"error", err,
				"event_type", ccr.Type(),
				"event_id", ccr.ID,
			)
			return err
		}
		// Log OriginalRequest details for debugging
		log.Debug(
			"[DEBUG] Creating CurrencyConverted event",
			"original_request_type", fmt.Sprintf("%T", ccr.OriginalRequest),
			"original_request_nil", ccr.OriginalRequest == nil,
			"transaction_id", ccr.TransactionID,
		)

		cc := events.NewCurrencyConverted(
			ccr,
			func(cc *events.CurrencyConverted) {
				cc.ConvertedAmount = convertedMoney
				cc.ConversionInfo = convInfo
				cc.TransactionID = ccr.TransactionID
				// Ensure OriginalRequest is preserved
				cc.OriginalRequest = ccr.OriginalRequest
			},
		)

		log.Info(
			"🔄 [PROCESS] Conversion completed successfully",
			"amount", ccr.Amount,
			"to", convertedMoney,
		)
		log.Info("📤 Emitting ", "event_type", cc.Type(), "event_id", cc.ID)
		if err = bus.Emit(ctx, cc); err != nil {
			log.Error(
				"Failed to emit done",
				"error", err,
				"event_type", cc.Type(),
				"event_id", cc.ID,
			)
			return err
		}

		// Delegate the creation of the next event to the factory.
		nextEvent := factory.CreateNextEvent(cc)
		log.Info(
			"✅ Created next event",
			"event_type", nextEvent.Type(),
			"event_id", ccr.ID,
			"correlation_id", ccr.CorrelationID,
		)
		log.Debug(
			"[DEBUG] Next event details",
			"event", nextEvent,
			"event_type", fmt.Sprintf("%T", nextEvent),
			"correlation_id", ccr.CorrelationID,
		)
		log.Info("📤 Emitting ", "event_type", nextEvent.Type())
		// Emit the next event in the flow
		if err := bus.Emit(ctx, nextEvent); err != nil {
			log.Error(
				"Failed to emit next event",
				"error", err,
				"event_type", nextEvent.Type(),
				"event_id", ccr.ID,
				"correlation_id", ccr.CorrelationID,
			)
			return fmt.Errorf("failed to emit next event: %w", err)
		}

		log.Info(
			"✅ Successfully emitted next event",
			"event_type", nextEvent.Type(),
			"event_id", ccr.ID,
			"correlation_id", ccr.CorrelationID,
		)
		return nil
	}
}
