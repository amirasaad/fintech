package account

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	mon "github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

// OperationHandler defines the interface for handling account operations in the chain
type OperationHandler interface {
	Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error)
	SetNext(handler OperationHandler)
}

// OperationRequest contains all the data needed for account operations
type OperationRequest struct {
	UserID         uuid.UUID
	AccountID      uuid.UUID
	Amount         float64
	CurrencyCode   currency.Code
	Operation      OperationType
	Account        *account.Account
	Money          mon.Money
	ConvertedMoney mon.Money
	ConvInfo       *common.ConversionInfo
	Transaction    *account.Transaction
}

// OperationResponse contains the result of an account operation
type OperationResponse struct {
	Transaction *account.Transaction
	ConvInfo    *common.ConversionInfo
	Error       error
}

// BaseHandler provides common functionality for all handlers
type BaseHandler struct {
	next OperationHandler
}

// SetNext sets the next handler in the chain
func (h *BaseHandler) SetNext(handler OperationHandler) {
	h.next = handler
}

// Handle passes the request to the next handler in the chain
func (h *BaseHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	if h.next != nil {
		return h.next.Handle(ctx, req)
	}
	return &OperationResponse{}, nil
}

// AccountValidationHandler validates that the account exists and belongs to the user
type AccountValidationHandler struct {
	BaseHandler
	uow    repository.UnitOfWork
	logger *slog.Logger
}

// Handle validates the account and passes the request to the next handler
func (h *AccountValidationHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger.With("userID", req.UserID, "accountID", req.AccountID)

	repo, err := h.uow.AccountRepository()
	if err != nil {
		logger.Error("AccountValidationHandler failed: repository error", "error", err)
		return &OperationResponse{Error: err}, nil
	}

	acc, err := repo.Get(req.AccountID)
	if err != nil {
		logger.Error("AccountValidationHandler failed: account not found", "error", err)
		return &OperationResponse{Error: account.ErrAccountNotFound}, nil
	}

	if acc.UserID != req.UserID {
		logger.Error("AccountValidationHandler failed: user unauthorized", "accountUserID", acc.UserID)
		return &OperationResponse{Error: user.ErrUserUnauthorized}, nil
	}

	req.Account = acc
	logger.Info("AccountValidationHandler: account validated successfully")

	return h.BaseHandler.Handle(ctx, req)
}

// MoneyCreationHandler creates a Money object from the request amount and currency
type MoneyCreationHandler struct {
	BaseHandler
	logger *slog.Logger
}

// Handle creates a Money object and passes the request to the next handler
func (h *MoneyCreationHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger.With("amount", req.Amount, "currency", req.CurrencyCode)

	money, err := mon.NewMoney(req.Amount, req.CurrencyCode)
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
	logger := h.logger.With("fromCurrency", req.Money.Currency(), "toCurrency", req.Account.Currency)

	if req.Money.Currency() == req.Account.Currency {
		req.ConvertedMoney = req.Money
		logger.Info("CurrencyConversionHandler: no conversion needed")
		return h.BaseHandler.Handle(ctx, req)
	}

	convInfo, err := h.converter.Convert(req.Money.AmountFloat(), string(req.Money.Currency()), string(req.Account.Currency))
	if err != nil {
		logger.Error("CurrencyConversionHandler failed: conversion error", "error", err)
		return &OperationResponse{Error: err}, nil
	}

	convertedMoney, err := mon.NewMoney(convInfo.ConvertedAmount, req.Account.Currency)
	if err != nil {
		logger.Error("CurrencyConversionHandler failed: converted money creation error", "error", err)
		return &OperationResponse{Error: err}, nil
	}

	req.ConvertedMoney = convertedMoney
	req.ConvInfo = convInfo
	logger.Info("CurrencyConversionHandler: conversion completed", "rate", convInfo.ConversionRate)

	return h.BaseHandler.Handle(ctx, req)
}

// DomainOperationHandler executes the domain operation (deposit/withdraw)
type DomainOperationHandler struct {
	BaseHandler
	logger *slog.Logger
}

// Handle executes the domain operation and passes the request to the next handler
func (h *DomainOperationHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger.With("operation", req.Operation)

	var tx *account.Transaction
	var err error

	switch req.Operation {
	case OperationDeposit:
		tx, err = req.Account.Deposit(req.UserID, req.ConvertedMoney)
	case OperationWithdraw:
		tx, err = req.Account.Withdraw(req.UserID, req.ConvertedMoney)
	default:
		err = fmt.Errorf("unsupported operation: %s", req.Operation)
	}

	if err != nil {
		logger.Error("DomainOperationHandler failed: domain operation error", "error", err)
		return &OperationResponse{Error: err}, nil
	}

	req.Transaction = tx
	logger.Info("DomainOperationHandler: domain operation completed", "transactionID", tx.ID)

	return h.BaseHandler.Handle(ctx, req)
}

// PersistenceHandler handles the persistence of account and transaction changes
type PersistenceHandler struct {
	BaseHandler
	uow    repository.UnitOfWork
	logger *slog.Logger
}

// Handle persists the changes and returns the final response
func (h *PersistenceHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger.With("transactionID", req.Transaction.ID)

	// Store conversion info if conversion occurred
	if req.ConvInfo != nil {
		req.Transaction.OriginalAmount = &req.ConvInfo.OriginalAmount
		req.Transaction.OriginalCurrency = &req.ConvInfo.OriginalCurrency
		req.Transaction.ConversionRate = &req.ConvInfo.ConversionRate
		logger.Info("PersistenceHandler: conversion info stored")
	}
	if err := h.uow.Do(ctx, func(uow repository.UnitOfWork) error {

		// Update account using type-safe method
		repo, err := uow.AccountRepository()

		if err != nil {
			logger.Error("PersistenceHandler failed: AccountRepository error", "error", err)
			return err
		}

		if err = repo.Update(req.Account); err != nil {
			logger.Error("PersistenceHandler failed: account update error", "error", err)
			return err
		}

		// Create transaction using type-safe method
		txRepo, err := uow.TransactionRepository()
		if err != nil {
			logger.Error("PersistenceHandler failed: TransactionRepository error", "error", err)
			return err
		}

		if err = txRepo.Create(req.Transaction); err != nil {
			logger.Error("PersistenceHandler failed: transaction create error", "error", err)
			return err
		}
		return err
	}); err != nil {
		return &OperationResponse{Error: err}, nil
	}

	logger.Info("PersistenceHandler: persistence completed successfully")

	return &OperationResponse{
		Transaction: req.Transaction,
		ConvInfo:    req.ConvInfo,
	}, nil

}

// ChainBuilder builds the operation chain
type ChainBuilder struct {
	uow       repository.UnitOfWork
	converter mon.CurrencyConverter
	logger    *slog.Logger
}

// NewChainBuilder creates a new chain builder
func NewChainBuilder(uow repository.UnitOfWork, converter mon.CurrencyConverter, logger *slog.Logger) *ChainBuilder {
	return &ChainBuilder{
		uow:       uow,
		converter: converter,
		logger:    logger,
	}
}

// BuildOperationChain builds and returns the complete operation chain
func (b *ChainBuilder) BuildOperationChain() OperationHandler {
	// Create handlers
	accountValidation := &AccountValidationHandler{
		uow:    b.uow,
		logger: b.logger,
	}

	moneyCreation := &MoneyCreationHandler{
		logger: b.logger,
	}

	currencyConversion := &CurrencyConversionHandler{
		converter: b.converter,
		logger:    b.logger,
	}

	domainOperation := &DomainOperationHandler{
		logger: b.logger,
	}

	persistence := &PersistenceHandler{
		uow:    b.uow,
		logger: b.logger,
	}

	// Chain them together
	accountValidation.SetNext(moneyCreation)
	moneyCreation.SetNext(currencyConversion)
	currencyConversion.SetNext(domainOperation)
	domainOperation.SetNext(persistence)

	return accountValidation
}
