// Package conversion handles currency conversion events and logic.
package conversion

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

// Handler processes CurrencyConversionRequested events, performs conversion, and emits CurrencyConversionDone.
func Handler(bus eventbus.EventBus, converter money.CurrencyConverter, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		logger := logger.With(
			"handler", "Handler",
			"event_type", e.EventType(),
		)
		logger.Info("received event", "event", e)

		ce, ok := e.(events.CurrencyConversionRequested)
		if !ok {
			logger.Error("unexpected event", "event", e)
			return
		}

		convInfo, err := converter.Convert(ce.Amount.AmountFloat(), ce.Amount.Currency().String(), ce.TargetCurrency)
		if err != nil {
			logger.Error("converter err", "error", err)
			return
		}

		// Perform conversion (stubbed for now)
		converted, err := money.New(convInfo.ConvertedAmount, currency.Code(ce.TargetCurrency))
		if err != nil {
			logger.Error("failed to process amount", "error", err)
			return
		}

		logger.Info("conversion done", "converted_money", converted)

		_ = bus.Publish(ctx, events.CurrencyConversionDone{
			CurrencyConversionRequested: ce,
			ConvertedAmount:             converted,
			ConversionInfo:              convInfo,
		})
	}
}
