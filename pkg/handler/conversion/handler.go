// Package conversion handles currency conversion events and logic.
package conversion

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/google/uuid"
)

// Handler processes ConversionRequestedEvent and delegates to a flow-specific factory to create the next event.
func Handler(
	bus eventbus.Bus,
	converter money.CurrencyConverter,
	logger *slog.Logger,
	factories map[string]EventFactory,
) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "ConversionHandler", "event_type", e.Type())
		log.Info("🟢 [START] Received event", "event", e)

		log.Debug("[DEBUG] Event type received", "type", fmt.Sprintf("%T", e))
		cre, ok := e.(*events.ConversionRequestedEvent)
		if !ok {
			log.Debug("🚫 [SKIP] Skipping: unexpected event type", "event", e)
			return nil
		}

		log.Debug("[DEBUG] ConversionRequestedEvent details", "event", cre)

		if cre.TransactionID == uuid.Nil {
			log.Error("Transaction ID is nil, discarding event", "event", cre)
			return errors.New("invalid transaction ID")
		}

		convInfo, err := converter.Convert(cre.FromAmount.AmountFloat(), cre.FromAmount.Currency().String(), cre.ToCurrency)
		if err != nil {
			log.Error("❌ [ERROR] Currency conversion failed", "error", err, "from_amount", cre.FromAmount, "to_currency", cre.ToCurrency)
			return err
		}

		log.Debug("[DEBUG] ConversionInfo", "convInfo", convInfo)

		convertedMoney, err := money.New(convInfo.ConvertedAmount, currency.Code(convInfo.ConvertedCurrency))
		if err != nil {
			log.Error("❌ [ERROR] Failed to create converted money object", "error", err, "convInfo", convInfo)
			return err
		}

		log.Info("🔄 [PROCESS] Conversion completed successfully", "from", cre.FromAmount, "to", convertedMoney)

		// Use the factory map to get the correct event factory for the flow type.
		factory, found := factories[cre.FlowType]
		if !found {
			log.Warn("Unknown flow type in ConversionRequestedEvent, discarding", "flow_type", cre.FlowType)
			return nil // Or return an error if this should be a hard failure
		}

		// Delegate the creation of the next event to the factory.
		nextEvent, err := factory.CreateNextEvent(cre, convInfo, convertedMoney)
		if err != nil {
			log.Error("❌ [ERROR] Failed to create next event", "error", err, "flow_type", cre.FlowType, "cre", cre, "convInfo", convInfo, "convertedMoney", convertedMoney)
			return err
		}

		if nextEvent == nil {
			log.Error("[ERROR] Factory returned nil nextEvent", "cre", cre, "convInfo", convInfo, "convertedMoney", convertedMoney)
			return errors.New("factory returned nil event")
		}

		log.Debug("[DEBUG] Next event to emit", "event", nextEvent)
		log.Info("📤 [EMIT] Emitting next event in flow", "event_type", nextEvent.Type(), "correlation_id", cre.CorrelationID.String())
		err = bus.Emit(ctx, nextEvent)
		if err != nil {
			log.Error("[ERROR] Failed to emit next event", "error", err, "event", nextEvent)
			return err
		}
		log.Debug("[DEBUG] Successfully emitted next event", "event_type", nextEvent.Type())
		return nil
	}
}
