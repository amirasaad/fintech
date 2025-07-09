// Package service provides business logic for interacting with domain entities such as accounts and transactions.
// It defines the AccountService struct and its methods for creating accounts, depositing and withdrawing funds,
// retrieving account details, listing transactions, and checking account balances.
package service

import (
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"

	"log/slog"

	"github.com/google/uuid"
)

// AccountService provides methods to interact with accounts and transactions using a unit of work pattern.
type AccountService struct {
	uowFactory func() (repository.UnitOfWork, error)
	converter  domain.CurrencyConverter
	logger     *slog.Logger
}

// NewAccountService creates a new AccountService with the given UnitOfWork factory, CurrencyConverter, and logger.
func NewAccountService(uowFactory func() (repository.UnitOfWork, error), converter domain.CurrencyConverter, logger *slog.Logger) *AccountService {
	return &AccountService{
		uowFactory: uowFactory,
		converter:  converter,
		logger:     logger,
	}
}

// CreateAccount creates a new account and persists it using the repository.
// Returns the created account or an error if the operation fails.
func (s *AccountService) CreateAccount(
	userID uuid.UUID,
) (a *domain.Account, err error) {
	uow, err := s.uowFactory()
	if err != nil {
		a = nil
		return
	}
	err = uow.Begin()
	if err != nil {
		a = nil
		return
	}

	a = domain.NewAccount(userID)
	repo, err := uow.AccountRepository()
	if err != nil {
		_ = uow.Rollback()
		a = nil
		return
	}
	err = repo.Create(a)
	if err != nil {
		_ = uow.Rollback()
		a = nil
		return
	}

	err = uow.Commit()
	if err != nil {
		_ = uow.Rollback()
		a = nil
		return
	}
	return
}

func (s *AccountService) CreateAccountWithCurrency(
	userID uuid.UUID,
	currencyCode currency.Code) (account *domain.Account, err error) {
	uow, err := s.uowFactory()
	if err != nil {
		account = nil
		return
	}
	err = uow.Begin()
	if err != nil {
		account = nil
		return
	}

	account, err = domain.NewAccountWithCurrency(userID, currencyCode)
	if err != nil {
		return
	}

	repo, err := uow.AccountRepository()
	if err != nil {
		_ = uow.Rollback()
		account = nil
		return
	}
	err = repo.Create(account)
	if err != nil {
		_ = uow.Rollback()
		account = nil
		return
	}

	err = uow.Commit()
	if err != nil {
		_ = uow.Rollback()
		account = nil
		return
	}
	return
}

// Deposit adds funds to the specified account and creates a transaction record.
// Returns the transaction or an error if the operation fails.
func (s *AccountService) Deposit(
	userID, accountID uuid.UUID,
	amount float64,
	currencyCode currency.Code,
) (tx *domain.Transaction, convInfo *domain.ConversionInfo, err error) {
	logger := s.logger
	logger.Info("Deposit started", "userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode)
	money, err := domain.NewMoney(amount, currencyCode)
	if err != nil {
		logger.Error("Deposit failed: invalid money", "error", err)
		tx = nil
		return
	}
	uow, err := s.uowFactory()
	if err != nil {
		logger.Error("Deposit failed: uowFactory error", "error", err)
		tx = nil
		return
	}
	err = uow.Begin()
	if err != nil {
		logger.Error("Deposit failed: uow.Begin error", "error", err)
		tx = nil
		return
	}

	repo, err := uow.AccountRepository()
	if err != nil {
		logger.Error("Deposit failed: AccountRepository error", "error", err)
		_ = uow.Rollback()
		tx = nil
		return
	}
	a, err := repo.Get(accountID)
	if err != nil {
		logger.Error("Deposit failed: account not found", "error", err)
		_ = uow.Rollback()
		tx = nil
		err = domain.ErrAccountNotFound
		return
	}

	if string(money.Currency()) != string(a.Currency) {
		logger.Info("Deposit: currency conversion needed", "from", money.Currency(), "to", a.Currency)
		convInfo, err = s.converter.Convert(money.AmountFloat(), string(money.Currency()), string(a.Currency))
		if err != nil {
			logger.Error("Deposit failed: currency conversion error", "error", err)
			_ = uow.Rollback()
			tx = nil
			return
		}
		money, err = domain.NewMoney(convInfo.ConvertedAmount, a.Currency)
		if err != nil {
			logger.Error("Deposit failed: new money after conversion error", "error", err)
			_ = uow.Rollback()
			tx = nil
			return
		}
	}

	tx, err = a.Deposit(userID, money)
	if err != nil {
		logger.Error("Deposit failed: domain deposit error", "error", err)
		_ = uow.Rollback()
		tx = nil
		return
	}

	if convInfo != nil {
		logger.Info("Deposit: conversion info stored", "originalAmount", convInfo.OriginalAmount, "originalCurrency", convInfo.OriginalCurrency, "conversionRate", convInfo.ConversionRate)
		tx.OriginalAmount = &convInfo.OriginalAmount
		tx.OriginalCurrency = &convInfo.OriginalCurrency
		tx.ConversionRate = &convInfo.ConversionRate
	}

	err = repo.Update(a)
	if err != nil {
		logger.Error("Deposit failed: repo update error", "error", err)
		_ = uow.Rollback()
		tx = nil
		return
	}

	txRepo, err := uow.TransactionRepository()
	if err != nil {
		logger.Error("Deposit failed: TransactionRepository error", "error", err)
		_ = uow.Rollback()
		tx = nil
		return
	}
	err = txRepo.Create(tx)
	if err != nil {
		logger.Error("Deposit failed: transaction create error", "error", err)
		_ = uow.Rollback()
		tx = nil
		return
	}

	err = uow.Commit()
	if err != nil {
		logger.Error("Deposit failed: commit error", "error", err)
		_ = uow.Rollback()
		tx = nil
		return
	}

	logger.Info("Deposit successful", "userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode, "transactionID", tx.ID)
	return
}

// Withdraw removes funds from the specified account and creates a transaction record.
// Returns the transaction, conversion info (if any), and error if the operation fails.
func (s *AccountService) Withdraw(
	userID, accountID uuid.UUID,
	amount float64,
	currencyCode currency.Code,
) (
	tx *domain.Transaction,
	convInfo *domain.ConversionInfo,
	err error,
) {
	logger := s.logger
	logger.Info("Withdraw started", "userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode)
	money, err := domain.NewMoney(amount, currencyCode)
	if err != nil {
		logger.Error("Withdraw failed: invalid money", "error", err)
		return nil, nil, err
	}
	uow, err := s.uowFactory()
	if err != nil {
		logger.Error("Withdraw failed: uowFactory error", "error", err)
		return nil, nil, err
	}
	err = uow.Begin()
	if err != nil {
		logger.Error("Withdraw failed: uow.Begin error", "error", err)
		return nil, nil, err
	}

	repo, err := uow.AccountRepository()
	if err != nil {
		logger.Error("Withdraw failed: AccountRepository error", "error", err)
		_ = uow.Rollback()
		return nil, nil, err
	}
	a, err := repo.Get(accountID)
	if err != nil {
		logger.Error("Withdraw failed: account not found", "error", err)
		_ = uow.Rollback()
		return nil, nil, domain.ErrAccountNotFound
	}

	if string(money.Currency()) != string(a.Currency) {
		logger.Info("Withdraw: currency conversion needed", "from", money.Currency(), "to", a.Currency)
		convInfo, err = s.converter.Convert(money.AmountFloat(), string(money.Currency()), string(a.Currency))
		if err != nil {
			logger.Error("Withdraw failed: currency conversion error", "error", err)
			_ = uow.Rollback()
			return
		}

		money, err = domain.NewMoney(convInfo.ConvertedAmount, a.Currency)
		if err != nil {
			logger.Error("Withdraw failed: new money after conversion error", "error", err)
			_ = uow.Rollback()
			return nil, nil, err
		}
	}

	tx, err = a.Withdraw(userID, money)
	if err != nil {
		logger.Error("Withdraw failed: domain withdraw error", "error", err)
		_ = uow.Rollback()
		return nil, nil, err
	}

	if convInfo != nil {
		logger.Info("Withdraw: conversion info stored", "originalAmount", convInfo.OriginalAmount, "originalCurrency", convInfo.OriginalCurrency, "conversionRate", convInfo.ConversionRate)
		tx.OriginalAmount = &convInfo.OriginalAmount
		tx.OriginalCurrency = &convInfo.OriginalCurrency
		tx.ConversionRate = &convInfo.ConversionRate
	}

	err = repo.Update(a)
	if err != nil {
		logger.Error("Withdraw failed: repo update error", "error", err)
		_ = uow.Rollback()
		return nil, nil, err
	}

	txRepo, err := uow.TransactionRepository()
	if err != nil {
		logger.Error("Withdraw failed: TransactionRepository error", "error", err)
		_ = uow.Rollback()
		return nil, nil, err
	}
	err = txRepo.Create(tx)
	if err != nil {
		logger.Error("Withdraw failed: transaction create error", "error", err)
		_ = uow.Rollback()
		return nil, nil, err
	}

	err = uow.Commit()
	if err != nil {
		logger.Error("Withdraw failed: commit error", "error", err)
		_ = uow.Rollback()
		return nil, nil, err
	}

	logger.Info("Withdraw successful", "userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode, "transactionID", tx.ID)
	return tx, convInfo, nil
}

// GetAccount retrieves an account by its ID.
// Returns the account or an error if not found.
func (s *AccountService) GetAccount(
	userID, accountID uuid.UUID,
) (a *domain.Account, err error) {
	uow, err := s.uowFactory()
	if err != nil {
		a = nil
		return
	}

	repo, err := uow.AccountRepository()
	if err != nil {
		a = nil
		return
	}
	a, err = repo.Get(accountID)
	if err != nil {
		a = nil
		err = domain.ErrAccountNotFound
		return
	}
	if a.UserID != userID {
		a = nil
		err = domain.ErrUserUnauthorized
		return
	}
	return
}

// GetTransactions retrieves all transactions for a given account ID.
// Returns a slice of transactions or an error if the operation fails.
func (s *AccountService) GetTransactions(
	userID, accountID uuid.UUID,
) (txs []*domain.Transaction, err error) {
	uow, err := s.uowFactory()
	if err != nil {
		txs = nil
		return
	}

	txRepo, err := uow.TransactionRepository()
	if err != nil {
		txs = nil
		return
	}
	txs, err = txRepo.List(userID, accountID)
	if err != nil {
		txs = nil
		return
	}

	return
}

// GetBalance retrieves the current balance of the specified account.
// Returns the balance as a float64 or an error if the operation fails.
func (s *AccountService) GetBalance(
	userID, accountID uuid.UUID,
) (balance float64, err error) {
	uow, err := s.uowFactory()
	if err != nil {
		return
	}

	repo, err := uow.AccountRepository()
	if err != nil {
		return
	}
	a, err := repo.Get(accountID)
	if err != nil {
		return
	}

	balance, err = a.GetBalance(userID)
	return
}
