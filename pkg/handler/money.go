package handler

import (
	"context"
	"log/slog"

	mon "github.com/amirasaad/fintech/pkg/domain/money"
)

// MoneyCreationHandler creates a Money object from the request amount and currency
type MoneyCreationHandler struct {
	BaseHandler
	logger *slog.Logger
}

// Handle creates a Money object and passes the request to the next handler
func (h *MoneyCreationHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger.With("amount", req.Amount, "currency", req.CurrencyCode)

	money, err := mon.New(req.Amount, req.CurrencyCode)
	if err != nil {
		logger.Error("MoneyCreationHandler failed: invalid money", "error", err)
		return &OperationResponse{Error: err}, nil
	}

	req.Money = money
	logger.Info("MoneyCreationHandler: money created successfully")

	return h.BaseHandler.Handle(ctx, req)
}

// CurrencyConversionHandler handles currency conversion if needed
type CurrencyConversionHandler struct {
	BaseHandler
	converter mon.CurrencyConverter
	logger    *slog.Logger
}

// Handle converts currency if needed and passes the request to the next handler
func (h *CurrencyConversionHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger.With("fromCurrency", req.Money.Currency(), "toCurrency", req.Account.Balance.Currency())

	if req.Money.Currency() == req.Account.Balance.Currency() {
		req.ConvertedMoney = req.Money
		logger.Info("CurrencyConversionHandler: no conversion needed")
		return h.BaseHandler.Handle(ctx, req)
	}

	convInfo, err := h.converter.Convert(req.Money.AmountFloat(), string(req.Money.Currency()), string(req.Account.Balance.Currency()))
	if err != nil {
		logger.Error("CurrencyConversionHandler failed: conversion error", "error", err)
		return &OperationResponse{Error: err}, nil
	}

	convertedMoney, err := mon.New(convInfo.ConvertedAmount, req.Account.Balance.Currency())
	if err != nil {
		logger.Error("CurrencyConversionHandler failed: converted money creation error", "error", err)
		return &OperationResponse{Error: err}, nil
	}

	req.ConvertedMoney = convertedMoney
	if req.Operation == OperationTransfer {
		req.ConvInfoOut = convInfo
		req.ConvInfoIn = convInfo
	} else {
		req.ConvInfo = convInfo
	}
	logger.Info("CurrencyConversionHandler: conversion completed", "rate", convInfo.ConversionRate)

	return h.BaseHandler.Handle(ctx, req)
}
