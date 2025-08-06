// Package conversion handles currency conversion events and logic.
package conversion

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

// HandleRequested processes ConversionRequestedEvent and
// delegates to a flow-specific factory to create the next event.
func HandleRequested(
	bus eventbus.Bus,
	converter money.CurrencyConverter,
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

		// If no conversion is needed, create the next event directly
		if ccr.Amount.IsCurrency(ccr.To.String()) {
			log.Debug(
				"No conversion needed, creating CurrencyConverted event",
				"original_request_type", fmt.Sprintf("%T", ccr.OriginalRequest),
				"original_request_nil", ccr.OriginalRequest == nil,
				"transaction_id", ccr.TransactionID,
			)

			// Create a CurrencyConverted event with the same amount
			cc := events.NewCurrencyConverted(
				ccr,
				func(cc *events.CurrencyConverted) {
					cc.ConvertedAmount = ccr.Amount
					cc.TransactionID = ccr.TransactionID
					// Ensure OriginalRequest is preserved
					cc.OriginalRequest = ccr.OriginalRequest
				},
			)

			// Use the factory to create the next event in the flow
			nextEvent := factory.CreateNextEvent(cc)
			if nextEvent == nil {
				log.Error(
					"Factory returned nil next event for non-converted amount")
				return errors.New("factory returned nil event")
			}

			log.Info(
				"🔄 [PROCESS] No conversion needed, proceeding to next event",
				"amount", ccr.Amount,
				"currency", ccr.To,
				"event_type", nextEvent.Type(),
				"correlation_id", ccr.CorrelationID,
			)

			return bus.Emit(ctx, nextEvent)
		}

		convInfo, err := converter.Convert(
			ccr.Amount.AmountFloat(),
			ccr.Amount.Currency().String(),
			ccr.To.String())
		if err != nil {
			log.Error(
				"Currency conversion failed",
				"error", err,
				"amount", ccr.Amount,
				"to_currency", ccr.To,
			)
			return err
		}

		log.Debug(
			"[DEBUG] ConversionInfo",
			"convInfo", convInfo,
		)

		convertedMoney, err := money.New(
			convInfo.ConvertedAmount,
			currency.Code(
				convInfo.ConvertedCurrency,
			),
		)
		if err != nil {
			log.Error(
				"Failed to create converted money object",
				"error", err,
				"convInfo", convInfo,
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
