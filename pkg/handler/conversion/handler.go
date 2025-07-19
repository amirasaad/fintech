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
		log.Info("🟢 [START] Received event", "event", e)

		// Handle both old and new event types for backward compatibility
		var fromAmount money.Money
		var toCurrency string
		var requestID string
		var flowType string
		var originalEvent interface{}

		switch evt := e.(type) {
		case events.ConversionRequestedEvent:
			fromAmount = evt.FromAmount
			toCurrency = evt.ToCurrency
			requestID = evt.RequestID
			flowType = "unknown" // New events don't have flow type
		case events.ConversionRequested:
			fromAmount = evt.Amount
			toCurrency = evt.TargetCurrency
			requestID = evt.CorrelationID
			flowType = evt.FlowType
			originalEvent = evt.OriginalEvent
		default:
			log.Error("❌ [ERROR] Unexpected event type for conversion handler - should only receive ConversionRequestedEvent", "event", e)
			return
		}

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
			"request_id", requestID,
			"flow_type", flowType)

		// Publish specific business events based on flow type
		switch flowType {
		case "deposit":
			if orig, ok := originalEvent.(events.DepositValidatedEvent); ok {
				doneEvent := events.DepositConversionDoneEvent{
					ConversionDoneEvent: events.ConversionDoneEvent{
						EventID:    uuid.New().String(),
						FromAmount: fromAmount,
						ToAmount:   converted,
						RequestID:  requestID,
						Timestamp:  time.Now(),
					},
					UserID:    orig.UserID.String(),
					AccountID: orig.AccountID.String(),
				}
				if err := bus.Publish(ctx, doneEvent); err != nil {
					log.Error("❌ [ERROR] Failed to publish DepositConversionDoneEvent", "error", err)
					return
				}
				log.Info("📤 [EMIT] DepositConversionDoneEvent published", "event", doneEvent)
			}
		case "withdraw":
			if orig, ok := originalEvent.(events.WithdrawValidatedEvent); ok {
				doneEvent := events.WithdrawConversionDoneEvent{
					ConversionDoneEvent: events.ConversionDoneEvent{
						EventID:    uuid.New().String(),
						FromAmount: fromAmount,
						ToAmount:   converted,
						RequestID:  requestID,
						Timestamp:  time.Now(),
					},
					UserID:    orig.UserID.String(),
					AccountID: orig.AccountID.String(),
				}
				if err := bus.Publish(ctx, doneEvent); err != nil {
					log.Error("❌ [ERROR] Failed to publish WithdrawConversionDoneEvent", "error", err)
					return
				}
				log.Info("📤 [EMIT] WithdrawConversionDoneEvent published", "event", doneEvent)
			}
		case "transfer":
			if orig, ok := originalEvent.(events.TransferValidatedEvent); ok {
				doneEvent := events.TransferConversionDoneEvent{
					ConversionDoneEvent: events.ConversionDoneEvent{
						EventID:    uuid.New().String(),
						FromAmount: fromAmount,
						ToAmount:   converted,
						RequestID:  requestID,
						Timestamp:  time.Now(),
					},
					SenderUserID:    orig.SenderUserID.String(),
					SourceAccountID: orig.SourceAccountID.String(),
					TargetAccountID: orig.DestAccountID.String(),
				}
				if err := bus.Publish(ctx, doneEvent); err != nil {
					log.Error("❌ [ERROR] Failed to publish TransferConversionDoneEvent", "error", err)
					return
				}
				log.Info("📤 [EMIT] TransferConversionDoneEvent published", "event", doneEvent)
			}
		default:
			// Fallback to generic event for unknown flow types
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
			log.Info("📤 [EMIT] ConversionDoneEvent published (fallback)", "event", doneEvent)
		}
	}
}
