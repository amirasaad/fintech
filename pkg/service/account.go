// Package service provides business logic for interacting with domain entities such as accounts and transactions.
// It defines the AccountService struct and its methods for creating accounts, depositing and withdrawing funds,
// retrieving account details, listing transactions, and checking account balances.
package service

import (
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	mon "github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/repository"

	"log/slog"

	"math"

	"github.com/amirasaad/fintech/pkg/decorator"
	"github.com/google/uuid"
)

// AccountService provides business logic for account operations
type AccountService struct {
	uowFactory  func() (repository.UnitOfWork, error)
	converter   mon.CurrencyConverter
	logger      *slog.Logger
	transaction decorator.TransactionDecorator
}

// NewAccountService creates a new AccountService instance
func NewAccountService(
	uowFactory func() (repository.UnitOfWork, error),
	converter mon.CurrencyConverter,
	logger *slog.Logger,
) *AccountService {
	return &AccountService{
		uowFactory:  uowFactory,
		converter:   converter,
		logger:      logger,
		transaction: decorator.NewUnitOfWorkTransactionDecorator(uowFactory, logger),
	}
}

// CreateAccount creates a new account and persists it using the repository.
// Returns the created account or an error if the operation fails.
func (s *AccountService) CreateAccount(
	userID uuid.UUID,
) (a *account.Account, err error) {
	logger := s.logger.With("userID", userID)
	logger.Info("CreateAccount started")
	var aLocal *account.Account
	err = s.transaction.Execute(func() error {
		aLocal, err = account.New().WithUserID(userID).Build()
		if err != nil {
			return err
		}
		uow, err := s.uowFactory()
		if err != nil {
			logger.Error("CreateAccount failed: uowFactory error", "error", err)
			return err
		}
		repo, err := uow.AccountRepository()
		if err != nil {
			logger.Error("CreateAccount failed: AccountRepository error", "error", err)
			return err
		}
		if err = repo.Create(aLocal); err != nil {
			logger.Error("CreateAccount failed: repo create error", "error", err)
			return err
		}
		return nil
	})
	if err != nil {
		logger.Error("CreateAccount failed: transaction error", "error", err)
		return nil, err
	}
	a = aLocal
	logger.Info("CreateAccount successful", "accountID", a.ID)
	return
}

func (s *AccountService) CreateAccountWithCurrency(
	userID uuid.UUID,
	currencyCode currency.Code,
) (acct *account.Account, err error) {
	logger := s.logger.With("userID", userID, "currency", currencyCode)
	logger.Info("CreateAccountWithCurrency started")
	err = s.transaction.Execute(func() error {
		var createErr error
		acct, createErr = account.New().
			WithUserID(userID).
			WithCurrency(currencyCode).
			Build()
		if createErr != nil {
			logger.Error("CreateAccountWithCurrency failed: domain error", "error", createErr)
			return createErr
		}
		uow, err := s.uowFactory()
		if err != nil {
			logger.Error("CreateAccountWithCurrency failed: uowFactory error", "error", err)
			return err
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
		logger.Error("CreateAccountWithCurrency failed: transaction error", "error", err)
		return
	}
	logger.Info("CreateAccountWithCurrency successful", "accountID", acct.ID)
	return
}

// Deposit adds funds to the specified account and creates a transaction record.
// Returns the transaction or an error if the operation fails.
func (s *AccountService) Deposit(
	userID, accountID uuid.UUID,
	amount float64,
	currencyCode currency.Code,
) (tx *account.Transaction, convInfo *common.ConversionInfo, err error) {
	logger := s.logger.With("userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode)
	logger.Info("Deposit started")

	var txLocal *account.Transaction
	var convInfoLocal *common.ConversionInfo
	err = s.transaction.Execute(func() error {
		uow, err := s.uowFactory()
		if err != nil {
			logger.Error("Deposit failed: uowFactory error", "error", err)
			return err
		}
		repo, err := uow.AccountRepository()
		if err != nil {
			logger.Error("Deposit failed: AccountRepository error", "error", err)
			return err
		}
		a, err := repo.Get(accountID)
		if err != nil {
			logger.Error("Deposit failed: account not found", "error", err)
			return account.ErrAccountNotFound
		}
		amountDeposit, err := mon.NewMoney(amount, currencyCode)
		if err != nil {
			logger.Error("Deposit failed: invalid money", "error", err)
			return err
		}

		// Inline currency conversion logic
		if amountDeposit.Currency() == a.Currency {
			// No conversion needed
		} else {
			convInfoLocal, err = s.converter.Convert(amountDeposit.AmountFloat(), string(amountDeposit.Currency()), string(a.Currency))
			if err != nil {
				logger.Error("Deposit failed: currency conversion error", "error", err)
				return err
			}
			meta, metaErr := currency.Get(string(a.Currency))
			if metaErr != nil {
				logger.Error("Deposit failed: currency metadata error", "error", metaErr)
				return metaErr
			}
			factor := math.Pow10(meta.Decimals)
			rounded := math.Round(convInfoLocal.ConvertedAmount*factor) / factor
			amountDeposit, err = mon.NewMoney(rounded, a.Currency)
			if err != nil {
				logger.Error("Deposit failed: converted money creation error", "error", err)
				return err
			}
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
		logger.Error("Deposit failed: transaction error", "error", err)
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
	tx *account.Transaction,
	convInfo *common.ConversionInfo,
	err error,
) {
	logger := s.logger.With("userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode)
	logger.Info("Withdraw started")

	var txLocal *account.Transaction
	var convInfoLocal *common.ConversionInfo
	err = s.transaction.Execute(func() error {
		uow, err := s.uowFactory()
		if err != nil {
			logger.Error("Withdraw failed: uowFactory error", "error", err)
			return err
		}
		repo, err := uow.AccountRepository()
		if err != nil {
			logger.Error("Withdraw failed: AccountRepository error", "error", err)
			return err
		}
		a, err := repo.Get(accountID)
		if err != nil {
			logger.Error("Withdraw failed: account not found", "error", err)
			return account.ErrAccountNotFound
		}
		m, err := mon.NewMoney(amount, currencyCode)
		if err != nil {
			logger.Error("Withdraw failed: invalid money", "error", err)
			return err
		}

		// Inline currency conversion logic
		if m.Currency() == a.Currency {
			// No conversion needed
		} else {
			convInfoLocal, err = s.converter.Convert(m.AmountFloat(), string(m.Currency()), string(a.Currency))
			if err != nil {
				logger.Error("Withdraw failed: currency conversion error", "error", err)
				return err
			}
			meta, metaErr := currency.Get(string(a.Currency))
			if metaErr != nil {
				logger.Error("Withdraw failed: currency metadata error", "error", metaErr)
				return metaErr
			}
			factor := math.Pow10(meta.Decimals)
			rounded := math.Round(convInfoLocal.ConvertedAmount*factor) / factor
			m, err = mon.NewMoney(rounded, a.Currency)
			if err != nil {
				logger.Error("Withdraw failed: converted money creation error", "error", err)
				return err
			}
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
		logger.Error("Withdraw failed: transaction error", "error", err)
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
) (a *account.Account, err error) {
	logger := s.logger.With("userID", userID, "accountID", accountID)
	logger.Info("GetAccount started")
	uow, err := s.uowFactory()
	if err != nil {
		logger.Error("GetAccount failed: uowFactory error", "error", err)
		return
	}
	repo, err := uow.AccountRepository()
	if err != nil {
		logger.Error("GetAccount failed: AccountRepository error", "error", err)
		return
	}
	aLocal, err := repo.Get(accountID)
	if err != nil {
		logger.Error("GetAccount failed: account not found", "error", err)
		err = account.ErrAccountNotFound
		return
	}
	if aLocal.UserID != userID {
		logger.Error("GetAccount failed: user unauthorized", "accountUserID", aLocal.UserID)
		err = user.ErrUserUnauthorized
		return
	}
	a = aLocal
	logger.Info("GetAccount successful")
	return
}

// GetTransactions retrieves all transactions for a given account ID.
// Returns a slice of transactions or an error if the operation fails.
func (s *AccountService) GetTransactions(
	userID, accountID uuid.UUID,
) (txs []*account.Transaction, err error) {
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
