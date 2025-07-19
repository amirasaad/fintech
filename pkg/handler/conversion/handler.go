// Package conversion handles currency conversion events and logic.
package conversion

import (
	"context"
	"log/slog"

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
		cre, ok := e.(events.ConversionRequestedEvent)
		if !ok {
			log.Error("unexpected event type for conversion handler", "event", e)
			return
		}
		fromAmount := cre.FromAmount
		toCurrency := cre.ToCurrency
		requestID := cre.RequestID

		// Perform currency conversion
		convInfo, err := converter.Convert(fromAmount.AmountFloat(), fromAmount.Currency().String(), toCurrency)
		if err != nil {
			log.Error("❌ [ERROR] Currency conversion failed", "error", err, "from", fromAmount.Currency().String(), "to", toCurrency)
			return
		}

		// Create Money objects for original and converted amounts
		originalMoney, err := money.New(convInfo.OriginalAmount, currency.Code(convInfo.OriginalCurrency))
		if err != nil {
			log.Error("❌ [ERROR] Failed to create original money", "error", err)
			return
		}
		convertedMoney, err := money.New(convInfo.ConvertedAmount, currency.Code(convInfo.ConvertedCurrency))
		if err != nil {
			log.Error("❌ [ERROR] Failed to create converted money", "error", err)
			return
		}

		log.Info("🔄 [PROCESS] Conversion completed successfully",
			"from", originalMoney,
			"to", convertedMoney,
			"rate", convInfo.ConversionRate,
			"request_id", requestID)

		log.Info("[DEBUG] Currency conversion result", "original_amount", convInfo.OriginalAmount, "original_currency", convInfo.OriginalCurrency, "converted_amount", convInfo.ConvertedAmount, "converted_currency", convInfo.ConvertedCurrency, "rate", convInfo.ConversionRate)

		// Determine flow type and emit the correct event
		flowType := cre.FlowType

		if cre.ToCurrency == "" {
			log.Error("[ERROR] ConversionRequestedEvent has empty ToCurrency", "event", cre)
		}

		switch flowType {
		case "deposit":
			depositEvent := events.DepositConversionDoneEvent{
				DepositValidatedEvent: events.DepositValidatedEvent{
					DepositRequestedEvent: events.DepositRequestedEvent{
						FlowEvent: cre.FlowEvent,
						ID:        cre.ID,
						Amount:    cre.FromAmount,
						Source:    "deposit",
						Timestamp: cre.Timestamp,
					},
				},
				// ConversionDoneEvent: conversionDone,
			}
			log.Info("📤 [EMIT] Emitting DepositConversionDoneEvent", "event", depositEvent, "correlation_id", cre.CorrelationID.String())
			_ = bus.Publish(ctx, depositEvent)
		case "withdraw":
			withdrawEvent := events.WithdrawConversionDoneEvent{
				WithdrawValidatedEvent: events.WithdrawValidatedEvent{
					WithdrawRequestedEvent: events.WithdrawRequestedEvent{
						FlowEvent: cre.FlowEvent,
						ID:        cre.ID,
						Amount:    cre.FromAmount,
						Timestamp: cre.Timestamp,
					},
				},
				// ConversionDoneEvent: conversionDone,
			}
			log.Info("📤 [EMIT] Emitting WithdrawConversionDoneEvent", "event", withdrawEvent, "correlation_id", cre.CorrelationID.String())
			_ = bus.Publish(ctx, withdrawEvent)
		case "transfer":
			transferEvent := events.TransferConversionDoneEvent{
				TransferValidatedEvent: events.TransferValidatedEvent{
					TransferRequestedEvent: events.TransferRequestedEvent{
						FlowEvent:      cre.FlowEvent,
						ID:             cre.ID,
						Amount:         cre.FromAmount,
						Source:         "transfer",
						DestAccountID:  cre.AccountID, // or another field as appropriate
						ReceiverUserID: cre.UserID,    // or another field as appropriate
					},
				},
				// ConversionDoneEvent: conversionDone,
			}
			log.Info("📤 [EMIT] Emitting TransferConversionDoneEvent", "event", transferEvent, "correlation_id", cre.CorrelationID.String())
			_ = bus.Publish(ctx, transferEvent)
		case "conversion":
			conversionDone := events.ConversionDoneEvent{
				FlowEvent:  cre.FlowEvent,
				ID:         uuid.New(),
				FromAmount: originalMoney,
				ToAmount:   convertedMoney,
				RequestID:  cre.RequestID,
				Timestamp:  cre.Timestamp,
			}
			log.Info("📤 [EMIT] Emitting ConversionDoneEvent", "event", conversionDone, "correlation_id", cre.CorrelationID.String())
			_ = bus.Publish(ctx, conversionDone)
		default:
			log.Warn("Unknown flow type in ConversionRequestedEvent", "flow_type", flowType)
		}
	}
}
