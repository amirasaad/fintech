package account

import (
	"context"
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
	TransactionIn  *account.Transaction // For transfer operations
	// For transfer
	DestAccountID uuid.UUID
	DestAccount   *account.Account
	DestUserID    uuid.UUID
}

// OperationResponse contains the result of an account operation
// For transfers, both TransactionOut (from source) and TransactionIn (to dest) may be set.
type OperationResponse struct {
	Transaction    *account.Transaction // For single-op or outgoing transfer
	TransactionOut *account.Transaction // Outgoing (source) transaction for transfer
	TransactionIn  *account.Transaction // Incoming (dest) transaction for transfer
	ConvInfo       *common.ConversionInfo
	Error          error
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

// ValidationHandler validates that the account exists and belongs to the user
type ValidationHandler struct {
	BaseHandler
	uow    repository.UnitOfWork
	logger *slog.Logger
}

// Handle validates the account and passes the request to the next handler
func (h *ValidationHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger.With("userID", req.UserID, "accountID", req.AccountID)
	logger.Info("AccountValidationHandler: starting")

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
	req.ConvInfo = convInfo
	logger.Info("CurrencyConversionHandler: conversion completed", "rate", convInfo.ConversionRate)

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
		logger.Info("PersistenceHandler: conversion info available", "originalAmount", req.ConvInfo.OriginalAmount)
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

		if err = txRepo.Create(req.Transaction, req.ConvInfo); err != nil {
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

// ChainBuilder builds operation-specific chains
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

// BuildDepositChain builds a chain for deposit operations
func (b *ChainBuilder) BuildDepositChain() OperationHandler {
	validation := &ValidationHandler{
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
	domainOperation := &DepositOperationHandler{
		logger: b.logger,
	}
	persistence := &PersistenceHandler{
		uow:    b.uow,
		logger: b.logger,
	}

	// Chain them together
	validation.SetNext(moneyCreation)
	moneyCreation.SetNext(currencyConversion)
	currencyConversion.SetNext(domainOperation)
	domainOperation.SetNext(persistence)

	return validation
}

// BuildWithdrawChain builds a chain for withdraw operations
func (b *ChainBuilder) BuildWithdrawChain() OperationHandler {
	validation := &ValidationHandler{
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
	domainOperation := &WithdrawOperationHandler{
		logger: b.logger,
	}
	persistence := &PersistenceHandler{
		uow:    b.uow,
		logger: b.logger,
	}

	// Chain them together
	validation.SetNext(moneyCreation)
	moneyCreation.SetNext(currencyConversion)
	currencyConversion.SetNext(domainOperation)
	domainOperation.SetNext(persistence)

	return validation
}

// BuildTransferChain builds a chain for transfer operations
func (b *ChainBuilder) BuildTransferChain() OperationHandler {
	validation := &TransferValidationHandler{
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
	domainOperation := &TransferOperationHandler{
		logger: b.logger,
	}
	persistence := &TransferPersistenceHandler{
		uow:    b.uow,
		logger: b.logger,
	}

	// Chain them together
	validation.SetNext(moneyCreation)
	moneyCreation.SetNext(currencyConversion)
	currencyConversion.SetNext(domainOperation)
	domainOperation.SetNext(persistence)

	return validation
}

// DepositOperationHandler executes deposit domain operations
type DepositOperationHandler struct {
	BaseHandler
	logger *slog.Logger
}

// Handle executes the deposit domain operation
func (h *DepositOperationHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger.With("operation", "deposit")

	tx, err := req.Account.Deposit(req.UserID, req.ConvertedMoney)
	if err != nil {
		logger.Error("DepositOperationHandler failed: domain operation error", "error", err)
		return &OperationResponse{Error: err}, nil
	}

	req.Transaction = tx
	logger.Info("DepositOperationHandler: domain operation completed", "transactionID", tx.ID)

	return h.BaseHandler.Handle(ctx, req)
}

// WithdrawOperationHandler executes withdraw domain operations
type WithdrawOperationHandler struct {
	BaseHandler
	logger *slog.Logger
}

// Handle executes the withdraw domain operation
func (h *WithdrawOperationHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger.With("operation", "withdraw")

	tx, err := req.Account.Withdraw(req.UserID, req.ConvertedMoney)
	if err != nil {
		logger.Error("WithdrawOperationHandler failed: domain operation error", "error", err)
		return &OperationResponse{Error: err}, nil
	}

	req.Transaction = tx
	logger.Info("WithdrawOperationHandler: domain operation completed", "transactionID", tx.ID)

	return h.BaseHandler.Handle(ctx, req)
}

// TransferValidationHandler validates both source and destination accounts for transfers
type TransferValidationHandler struct {
	BaseHandler
	uow    repository.UnitOfWork
	logger *slog.Logger
}

// Handle validates both accounts and passes the request to the next handler
func (h *TransferValidationHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger.With("userID", req.UserID, "sourceAccountID", req.AccountID, "destAccountID", req.DestAccountID)
	logger.Info("TransferValidationHandler: starting")

	repo, err := h.uow.AccountRepository()
	if err != nil {
		logger.Error("TransferValidationHandler failed: repository error", "error", err)
		return &OperationResponse{Error: err}, nil
	}

	// Get and validate source account
	sourceAccount, err := repo.Get(req.AccountID)
	if err != nil {
		logger.Error("TransferValidationHandler failed: source account not found", "error", err)
		return &OperationResponse{Error: account.ErrAccountNotFound}, nil
	}

	if sourceAccount.UserID != req.UserID {
		logger.Error("TransferValidationHandler failed: user unauthorized for source account", "accountUserID", sourceAccount.UserID)
		return &OperationResponse{Error: user.ErrUserUnauthorized}, nil
	}

	// Get and validate destination account
	destAccount, err := repo.Get(req.DestAccountID)
	if err != nil {
		logger.Error("TransferValidationHandler failed: destination account not found", "error", err)
		return &OperationResponse{Error: account.ErrAccountNotFound}, nil
	}

	req.Account = sourceAccount
	req.DestAccount = destAccount
	logger.Info("TransferValidationHandler: both accounts validated successfully")

	return h.BaseHandler.Handle(ctx, req)
}

// TransferOperationHandler executes transfer domain operations
type TransferOperationHandler struct {
	BaseHandler
	logger *slog.Logger
}

// Handle executes the transfer domain operation
func (h *TransferOperationHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger.With("operation", "transfer")

	txIn, txOut, err := req.Account.Transfer(req.UserID, req.DestAccount, req.ConvertedMoney)
	if err != nil {
		logger.Error("TransferOperationHandler failed: domain operation error", "error", err)
		return &OperationResponse{Error: err}, nil
	}

	req.Transaction = txOut
	req.TransactionIn = txIn
	logger.Info("TransferOperationHandler: domain operation completed", "txOutID", txOut.ID, "txInID", txIn.ID)

	return h.BaseHandler.Handle(ctx, req)
}

// TransferPersistenceHandler handles persistence for transfer operations
type TransferPersistenceHandler struct {
	BaseHandler
	uow    repository.UnitOfWork
	logger *slog.Logger
}

// Handle persists transfer changes and returns the final response
func (h *TransferPersistenceHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	logger := h.logger.With("transactionID", req.Transaction.ID)

	if err := h.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		// Update both accounts
		repo, err := uow.AccountRepository()
		if err != nil {
			logger.Error("TransferPersistenceHandler failed: AccountRepository error", "error", err)
			return err
		}

		if err = repo.Update(req.Account); err != nil {
			logger.Error("TransferPersistenceHandler failed: source account update error", "error", err)
			return err
		}

		if err = repo.Update(req.DestAccount); err != nil {
			logger.Error("TransferPersistenceHandler failed: destination account update error", "error", err)
			return err
		}

		// Create both transaction records
		txRepo, err := uow.TransactionRepository()
		if err != nil {
			logger.Error("TransferPersistenceHandler failed: TransactionRepository error", "error", err)
			return err
		}

		if err = txRepo.Create(req.Transaction, req.ConvInfo); err != nil {
			logger.Error("TransferPersistenceHandler failed: outgoing transaction create error", "error", err)
			return err
		}

		if err = txRepo.Create(req.TransactionIn, nil); err != nil {
			logger.Error("TransferPersistenceHandler failed: incoming transaction create error", "error", err)
			return err
		}

		return nil
	}); err != nil {
		return &OperationResponse{Error: err}, nil
	}

	logger.Info("TransferPersistenceHandler: persistence completed successfully")

	return &OperationResponse{
		Transaction:    req.Transaction,
		TransactionOut: req.Transaction,
		TransactionIn:  req.TransactionIn,
		ConvInfo:       req.ConvInfo,
	}, nil
}
