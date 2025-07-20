// Package conversion handles currency conversion events and logic.
package conversion

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/google/uuid"
)

// Handler processes ConversionRequestedEvent and emits conversion done events for deposit/withdraw/transfer.
func Handler(bus eventbus.Bus, converter money.CurrencyConverter, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "ConversionHandler", "event_type", e.Type())
		log.Info("🟢 [START] Received event", "event", e)

		cre, ok := e.(*events.ConversionRequestedEvent)
		if !ok {
			log.Debug("🚫 [SKIP] Skipping: unexpected event type in ConversionHandler", "event", e)
			return nil
		}
		if cre.TransactionID == uuid.Nil {
			log.Error("Transaction ID is nil, aborting ConversionDoneEvent emission", "event", cre)
			return errors.New("invalid transaction ID")
		}
		fromAmount := cre.FromAmount
		toCurrency := cre.ToCurrency
		requestID := cre.RequestID

		// Perform currency conversion
		convInfo, err := converter.Convert(fromAmount.AmountFloat(), fromAmount.Currency().String(), toCurrency)
		if err != nil {
			log.Error("❌ [ERROR] Currency conversion failed", "error", err, "from", fromAmount.Currency().String(), "to", toCurrency)
			return err
		}

		// Create Money objects for original and converted amounts
		originalMoney, err := money.New(convInfo.OriginalAmount, currency.Code(convInfo.OriginalCurrency))
		if err != nil {
			log.Error("❌ [ERROR] Failed to create original money", "error", err)
			return err
		}
		convertedMoney, err := money.New(convInfo.ConvertedAmount, currency.Code(convInfo.ConvertedCurrency))
		if err != nil {
			log.Error("❌ [ERROR] Failed to create converted money", "error", err)
			return err
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
			conversionRate := convInfo.ConversionRate
			originalCurrency := originalMoney.Currency().String()
			convertedAmount := convertedMoney.AmountFloat()
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
				ConversionDoneEvent: events.ConversionDoneEvent{
					FlowEvent:        cre.FlowEvent,
					ID:               uuid.New(),
					FromAmount:       cre.FromAmount,
					ToAmount:         convertedMoney,
					RequestID:        cre.RequestID,
					TransactionID:    cre.TransactionID,
					Timestamp:        cre.Timestamp,
					ConversionRate:   conversionRate,
					OriginalCurrency: originalCurrency,
					ConvertedAmount:  convertedAmount,
				},
				TransactionID: cre.TransactionID,
			}
			log.Info("📤 [EMIT] Emitting DepositConversionDoneEvent", "event", depositEvent, "correlation_id", cre.CorrelationID.String())
			// After constructing ConversionDoneEvent and DepositConversionDoneEvent
			log.Info("[DEBUG] Conversion event construction",
				"from_currency", cre.FromAmount.Currency().String(),
				"to_currency", convertedMoney.Currency().String(),
				"original_currency", originalCurrency)
			return bus.Emit(ctx, depositEvent)
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
			return bus.Emit(ctx, withdrawEvent)
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
			return bus.Emit(ctx, transferEvent)
		case "conversion":
			conversionRate := convInfo.ConversionRate
			originalCurrency := originalMoney.Currency().String()
			convertedAmount := convertedMoney.AmountFloat()
			conversionDone := events.ConversionDoneEvent{
				FlowEvent:        cre.FlowEvent,
				ID:               uuid.New(),
				FromAmount:       cre.FromAmount,
				ToAmount:         convertedMoney,
				RequestID:        cre.RequestID,
				TransactionID:    cre.TransactionID,
				Timestamp:        cre.Timestamp,
				ConversionRate:   conversionRate,
				OriginalCurrency: originalCurrency,
				ConvertedAmount:  convertedAmount,
			}
			log.Info("📤 [EMIT] Emitting ConversionDoneEvent", "event", conversionDone, "correlation_id", cre.CorrelationID.String())
			return bus.Emit(ctx, conversionDone)
		default:
			log.Warn("Unknown flow type in ConversionRequestedEvent", "flow_type", flowType)
			return nil
		}
	}
}
