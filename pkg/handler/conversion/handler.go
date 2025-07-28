// Package conversion handles currency conversion events and logic.
package conversion

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/common"
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
) func(ctx context.Context, e common.Event) error {
	return func(ctx context.Context, e common.Event) error {
		log := logger.With("handler", "ConversionHandler", "event_type", e.Type())
		log.Info("🟢 [START] Received event", "event", e)

		log.Debug("[DEBUG] Event type received", "type", fmt.Sprintf("%T", e))
		cre, ok := e.(*events.ConversionRequestedEvent)
		if !ok {
			log.Debug("🚫 [SKIP] Skipping: unexpected event type", "event", e)
			return nil
		}

		log.Debug("[DEBUG] ConversionRequestedEvent details", "event", cre)
		// Use the factory map to get the correct event factory for the flow type.
		factory, found := factories[cre.FlowType]
		if !found {
			log.Warn("Unknown flow type in ConversionRequestedEvent, discarding", "flow_type", cre.FlowType)
			return nil // Or return an error if this should be a hard failure
		}

		if cre.Amount.IsCurrency(cre.To.String()) {
			if nextEvent, err := factory.CreateNextEvent(cre, nil, cre.Amount); err == nil {
				return bus.Emit(ctx, nextEvent)
			}

		}
		if cre.TransactionID == uuid.Nil {
			log.Error("Transaction ID is nil, discarding event", "event", cre)
			return errors.New("invalid transaction ID")
		}

		convInfo, err := converter.Convert(
			cre.Amount.AmountFloat(),
			cre.Amount.Currency().String(),
			cre.To.String())
		if err != nil {
			log.Error("❌ [ERROR] Currency conversion failed", "error", err, "amount", cre.Amount, "to_currency", cre.To)
			return err
		}

		log.Debug("[DEBUG] ConversionInfo", "convInfo", convInfo)

		convertedMoney, err := money.New(convInfo.ConvertedAmount, currency.Code(convInfo.ConvertedCurrency))
		if err != nil {
			log.Error("❌ [ERROR] Failed to create converted money object", "error", err, "convInfo", convInfo)
			return err
		}
		conversionDone := events.ConversionDoneEvent{
			ID:              uuid.New(),
			FlowEvent:       cre.FlowEvent,
			TransactionID:   cre.TransactionID,
			ConversionInfo:  convInfo,
			ConvertedAmount: convertedMoney,
			Timestamp:       time.Now(),
		}
		log.Info("🔄 [PROCESS] Conversion completed successfully", "amount", cre.Amount, "to", convertedMoney)
		log.Info("📤 [EMIT] Emitting conversion done ", "event_type", conversionDone)
		if err = bus.Emit(ctx, &conversionDone); err != nil {
			log.Error("[ERROR] Failed to emit conversion done", "error", err, "event", conversionDone)
			return err
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
		// Best practice: always use pointer events for emission
		log.Info("📤 [EMIT] Emitting next event in flow", "event_type", nextEvent.Type(), "event_pointer", fmt.Sprintf("%T", nextEvent), "correlation_id", cre.CorrelationID.String())
		log.Debug("[DEBUG] Type name of nextEvent before emit", "type_name", reflect.TypeOf(nextEvent).String())

		// Emit as pointer if not already
		return bus.Emit(ctx, nextEvent)
		// If nextEvent is a value, use &nextEvent
	}
}
