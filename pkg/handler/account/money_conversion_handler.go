package account

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

// MoneyConversionHandler handles MoneyCreatedEvent, performs currency conversion if needed, and publishes MoneyConvertedEvent.
func MoneyConversionHandler(bus eventbus.EventBus, converter money.CurrencyConverter) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		me, ok := e.(accountdomain.MoneyCreatedEvent)
		if !ok {
			return
		}
		origMoney := me.Money
		targetCurrency := me.TargetCurrency
		if string(origMoney.Currency()) == string(targetCurrency) {
			_ = bus.Publish(ctx, accountdomain.MoneyConvertedEvent{
				MoneyCreatedEvent: me,
				Money:             origMoney,
				ConversionInfo:    nil,
			})
			return
		}
		convInfo, err := converter.Convert(origMoney.AmountFloat(), string(origMoney.Currency()), string(targetCurrency))
		if err != nil {
			slog.Error("MoneyConversionHandler: conversion failed", "error", err)
			return
		}
		convertedMoney, err := money.New(convInfo.ConvertedAmount, targetCurrency)
		if err != nil {
			slog.Error("MoneyConversionHandler: failed to create converted money", "error", err)
			return
		}
		_ = bus.Publish(ctx, accountdomain.MoneyConvertedEvent{
			MoneyCreatedEvent: me,
			Money:             convertedMoney,
			ConversionInfo:    convInfo,
		})
	}
}
