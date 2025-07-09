// Package service provides business logic for interacting with domain entities such as accounts and transactions.
// It defines the AccountService struct and its methods for creating accounts, depositing and withdrawing funds,
// retrieving account details, listing transactions, and checking account balances.
package service

import (
	"github.com/amirasaad/fintech/pkg/currency"
	acc "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	mon "github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/repository"

	"log/slog"

	"github.com/google/uuid"
)

// AccountService provides methods to interact with accounts and transactions using a unit of work pattern.
type AccountService struct {
	uowFactory func() (repository.UnitOfWork, error)
	converter  mon.CurrencyConverter
	logger     *slog.Logger
}

// NewAccountService creates a new AccountService with the given UnitOfWork factory, CurrencyConverter, and logger.
func NewAccountService(uowFactory func() (repository.UnitOfWork, error), converter mon.CurrencyConverter, logger *slog.Logger) *AccountService {
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
) (a *acc.Account, err error) {
	logger := s.logger.With("userID", userID)
	logger.Info("CreateAccount started")
	var aLocal *acc.Account
	if err = s.withTransaction(func(uow repository.UnitOfWork) (err error) {
		aLocal, err = acc.New().WithUserID(userID).Build()
		if err != nil {
			return
		}
		repo, err := uow.AccountRepository()
		if err != nil {
			logger.Error("CreateAccount failed: AccountRepository error", "error", err)
			return
		}
		if err = repo.Create(aLocal); err != nil {
			logger.Error("CreateAccount failed: repo create error", "error", err)
			return
		}
		return
	}); err != nil {
		logger.Error("CreateAccount failed: withTransaction error", "error", err)
		return
	}
	a = aLocal
	logger.Info("CreateAccount successful", "accountID", a.ID)
	return
}

func (s *AccountService) CreateAccountWithCurrency(
	userID uuid.UUID,
	currencyCode currency.Code,
) (acct *acc.Account, err error) {
	logger := s.logger.With("userID", userID, "currency", currencyCode)
	logger.Info("CreateAccountWithCurrency started")
	err = s.withTransaction(func(uow repository.UnitOfWork) error {
		var createErr error
		acct, createErr = acc.New().
			WithUserID(userID).
			WithCurrency(currencyCode).
			Build()
		if createErr != nil {
			logger.Error("CreateAccountWithCurrency failed: domain error", "error", createErr)
			return createErr
		}
		repo, err := uow.AccountRepository()
		if err != nil {
			logger.Error("CreateAccountWithCurrency failed: AccountRepository error", "error", err)
			return err
		}
		if err = repo.Create(acct); err != nil {
			logger.Error("CreateAccountWithCurrency failed: repo create error", "error", err)
			return err
		}
		return nil
	})
	if err != nil {
		acct = nil
		logger.Error("CreateAccountWithCurrency failed: withTransaction error", "error", err)
		return
	}
	logger.Info("CreateAccountWithCurrency successful", "accountID", acct.ID)
	return
}

// handleCurrencyConversion encapsulates the conversion logic for Deposit and Withdraw
func (s *AccountService) handleCurrencyConversion(
	m mon.Money,
	target currency.Code,
) (
	convertedMoney mon.Money,
	convInfo *common.ConversionInfo,
	err error,
) {
	if m.Currency() == target {
		convertedMoney = m
		return
	}
	convInfo, err = s.converter.Convert(m.AmountFloat(), string(m.Currency()), string(target))
	if err != nil {
		return
	}
	convertedMoney, err = mon.NewMoney(convInfo.ConvertedAmount, target)
	if err != nil {
		return
	}
	return
}

// withTransaction abstracts transaction orchestration for the service layer.
func (s *AccountService) withTransaction(
	fn func(uow repository.UnitOfWork) error,
) (
	err error,
) {
	uow, err := s.uowFactory()
	if err != nil {
		return
	}
	if err = uow.Begin(); err != nil {
		return
	}
	var fnErr error
	defer func() {
		if fnErr != nil {
			_ = uow.Rollback()
		}
	}()
	fnErr = fn(uow)
	if fnErr != nil {
		err = fnErr
		return
	}
	err = uow.Commit()
	if err != nil {
		_ = uow.Rollback()
	}
	return
}

// Deposit adds funds to the specified account and creates a transaction record.
// Returns the transaction or an error if the operation fails.
func (s *AccountService) Deposit(
	userID, accountID uuid.UUID,
	amount float64,
	currencyCode currency.Code,
) (tx *acc.Transaction, convInfo *common.ConversionInfo, err error) {
	logger := s.logger.With("userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode)
	logger.Info("Deposit started")

	var txLocal *acc.Transaction
	var convInfoLocal *common.ConversionInfo
	err = s.withTransaction(func(uow repository.UnitOfWork) error {
		repo, err := uow.AccountRepository()
		if err != nil {
			logger.Error("Deposit failed: AccountRepository error", "error", err)
			return err
		}
		a, err := repo.Get(accountID)
		if err != nil {
			logger.Error("Deposit failed: account not found", "error", err)
			return acc.ErrAccountNotFound
		}
		amountDeposit, err := mon.NewMoney(amount, currencyCode)
		if err != nil {
			logger.Error("Deposit failed: invalid money", "error", err)
			return err
		}
		var convErr error
		amountDeposit, convInfoLocal, convErr = s.handleCurrencyConversion(amountDeposit, a.Currency)
		if convErr != nil {
			logger.Error("Deposit failed: currency conversion error", "error", convErr)
			return convErr
		}
		txLocal, err = a.Deposit(userID, amountDeposit)
		if err != nil {
			logger.Error("Deposit failed: domain deposit error", "error", err)
			return err
		}
		if convInfoLocal != nil {
			logger.Info(
				"Deposit: conversion info stored",
				"originalAmount", convInfoLocal.OriginalAmount,
				"originalCurrency", convInfoLocal.OriginalCurrency,
				"conversionRate", convInfoLocal.ConversionRate,
			)
			txLocal.OriginalAmount = &convInfoLocal.OriginalAmount
			txLocal.OriginalCurrency = &convInfoLocal.OriginalCurrency
			txLocal.ConversionRate = &convInfoLocal.ConversionRate
		}
		if err = repo.Update(a); err != nil {
			logger.Error("Deposit failed: repo update error", "error", err)
			return err
		}
		txRepo, err := uow.TransactionRepository()
		if err != nil {
			logger.Error("Deposit failed: TransactionRepository error", "error", err)
			return err
		}
		if err = txRepo.Create(txLocal); err != nil {
			logger.Error("Deposit failed: transaction create error", "error", err)
			return err
		}
		return nil
	})
	if err != nil {
		tx = nil
		convInfo = nil
		logger.Error("Deposit failed: withTransaction error", "error", err)
		return
	}
	tx = txLocal
	convInfo = convInfoLocal
	logger.Info("Deposit successful", "transactionID", tx.ID)
	return
}

// Withdraw removes funds from the specified account and creates a transaction record.
// Returns the transaction, conversion info (if any), and error if the operation fails.
func (s *AccountService) Withdraw(
	userID, accountID uuid.UUID,
	amount float64,
	currencyCode currency.Code,
) (
	tx *acc.Transaction,
	convInfo *common.ConversionInfo,
	err error,
) {
	logger := s.logger.With("userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode)
	logger.Info("Withdraw started")

	var txLocal *acc.Transaction
	var convInfoLocal *common.ConversionInfo
	err = s.withTransaction(func(uow repository.UnitOfWork) error {
		repo, err := uow.AccountRepository()
		if err != nil {
			logger.Error("Withdraw failed: AccountRepository error", "error", err)
			return err
		}
		a, err := repo.Get(accountID)
		if err != nil {
			logger.Error("Withdraw failed: account not found", "error", err)
			return acc.ErrAccountNotFound
		}
		m, err := mon.NewMoney(amount, currencyCode)
		if err != nil {
			logger.Error("Withdraw failed: invalid money", "error", err)
			return err
		}
		var convErr error
		m, convInfoLocal, convErr = s.handleCurrencyConversion(m, a.Currency)
		if convErr != nil {
			logger.Error("Withdraw failed: currency conversion error", "error", convErr)
			return convErr
		}
		txLocal, err = a.Withdraw(userID, m)
		if err != nil {
			logger.Error("Withdraw failed: domain withdraw error", "error", err)
			return err
		}
		if convInfoLocal != nil {
			logger.Info(
				"Withdraw: conversion info stored",
				"originalAmount", convInfoLocal.OriginalAmount,
				"originalCurrency", convInfoLocal.OriginalCurrency,
				"conversionRate", convInfoLocal.ConversionRate,
			)
			txLocal.OriginalAmount = &convInfoLocal.OriginalAmount
			txLocal.OriginalCurrency = &convInfoLocal.OriginalCurrency
			txLocal.ConversionRate = &convInfoLocal.ConversionRate
		}
		if err = repo.Update(a); err != nil {
			logger.Error("Withdraw failed: repo update error", "error", err)
			return err
		}
		txRepo, err := uow.TransactionRepository()
		if err != nil {
			logger.Error("Withdraw failed: TransactionRepository error", "error", err)
			return err
		}
		if err = txRepo.Create(txLocal); err != nil {
			logger.Error("Withdraw failed: transaction create error", "error", err)
			return err
		}
		return nil
	})
	if err != nil {
		tx = nil
		convInfo = nil
		logger.Error("Withdraw failed: withTransaction error", "error", err)
		return
	}
	tx = txLocal
	convInfo = convInfoLocal
	logger.Info("Withdraw successful", "transactionID", tx.ID)
	return
}

// GetAccount retrieves an account by its ID.
// Returns the account or an error if not found.
func (s *AccountService) GetAccount(
	userID, accountID uuid.UUID,
) (a *acc.Account, err error) {
	logger := s.logger.With("userID", userID, "accountID", accountID)
	logger.Info("GetAccount started")
	uow, err := s.uowFactory()
	if err != nil {
		a = nil
		logger.Error("GetAccount failed: uowFactory error", "error", err)
		return
	}
	repo, err := uow.AccountRepository()
	if err != nil {
		a = nil
		logger.Error("GetAccount failed: AccountRepository error", "error", err)
		return
	}
	a, err = repo.Get(accountID)
	if err != nil {
		a = nil
		logger.Error("GetAccount failed: account not found", "error", err)
		err = acc.ErrAccountNotFound
		return
	}
	if a.UserID != userID {
		a = nil
		logger.Error("GetAccount failed: user unauthorized", "accountUserID", a.UserID)
		err = user.ErrUserUnauthorized
		return
	}
	logger.Info("GetAccount successful")
	return
}

// GetTransactions retrieves all transactions for a given account ID.
// Returns a slice of transactions or an error if the operation fails.
func (s *AccountService) GetTransactions(
	userID, accountID uuid.UUID,
) (txs []*acc.Transaction, err error) {
	logger := s.logger.With("userID", userID, "accountID", accountID)
	logger.Info("GetTransactions started")
	uow, err := s.uowFactory()
	if err != nil {
		txs = nil
		logger.Error("GetTransactions failed: uowFactory error", "error", err)
		return
	}
	txRepo, err := uow.TransactionRepository()
	if err != nil {
		txs = nil
		logger.Error("GetTransactions failed: TransactionRepository error", "error", err)
		return
	}
	txs, err = txRepo.List(userID, accountID)
	if err != nil {
		txs = nil
		logger.Error("GetTransactions failed: repo list error", "error", err)
		return
	}
	logger.Info("GetTransactions successful", "count", len(txs))
	return
}

// GetBalance retrieves the current balance of the specified account.
// Returns the balance as a float64 or an error if the operation fails.
func (s *AccountService) GetBalance(
	userID, accountID uuid.UUID,
) (balance float64, err error) {
	logger := s.logger.With("userID", userID, "accountID", accountID)
	logger.Info("GetBalance started")
	uow, err := s.uowFactory()
	if err != nil {
		logger.Error("GetBalance failed: uowFactory error", "error", err)
		return
	}
	repo, err := uow.AccountRepository()
	if err != nil {
		logger.Error("GetBalance failed: AccountRepository error", "error", err)
		return
	}
	a, err := repo.Get(accountID)
	if err != nil {
		logger.Error("GetBalance failed: account not found", "error", err)
		return
	}
	balance, err = a.GetBalance(userID)
	if err != nil {
		logger.Error("GetBalance failed: domain error", "error", err)
		return
	}
	logger.Info("GetBalance successful", "balance", balance)
	return
}
