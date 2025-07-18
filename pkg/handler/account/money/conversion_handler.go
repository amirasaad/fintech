package money

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

// MoneyConversionHandler handles MoneyCreatedEvent, performs currency conversion if needed, and publishes MoneyConvertedEvent.
func MoneyConversionHandler(bus eventbus.EventBus, converter money.CurrencyConverter) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		me, ok := e.(events.MoneyCreatedEvent)
		if !ok {
			return
		}
		origMoney, err := money.NewMoneyFromSmallestUnit(me.Amount, currency.Code(me.Currency))
		if err != nil {
			slog.Error("MoneyConversionHandler: conversion failed", "error", err)
			return
		}
		targetCurrency := me.TargetCurrency
		if origMoney.IsCurrency(targetCurrency) {
			_ = bus.Publish(ctx, events.MoneyConvertedEvent{
				MoneyCreatedEvent: me,
				Amount:            origMoney.Amount(),
				Currency:          origMoney.Currency().String(),
				ConversionInfo:    nil,
				TransactionID:     me.TransactionID,
			})
			return
		}
		convInfo, err := converter.Convert(origMoney.AmountFloat(), string(origMoney.Currency()), string(targetCurrency))
		if err != nil {
			slog.Error("MoneyConversionHandler: conversion failed", "error", err)
			return
		}
		convertedMoney, err := money.New(convInfo.ConvertedAmount, currency.Code(targetCurrency))
		if err != nil {
			slog.Error("MoneyConversionHandler: failed to create converted money", "error", err)
			return
		}
		_ = bus.Publish(ctx, events.MoneyConvertedEvent{
			MoneyCreatedEvent: me,
			Amount:            convertedMoney.Amount(),
			Currency:          convertedMoney.Currency().String(),
			ConversionInfo:    convInfo,
			TransactionID:     me.TransactionID,
		})
	}
}
