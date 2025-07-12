// Package service provides business logic for interacting with domain entities such as accounts and transactions.
// It defines the AccountService struct and its methods for creating accounts, depositing and withdrawing funds,
// retrieving account details, listing transactions, and checking account balances.
//
// The service layer follows clean architecture principles and uses the decorator pattern for transaction management.
// All business operations are wrapped with automatic transaction handling, error recovery, and structured logging.
package service

import (
	"context"
	"errors"
	"log/slog"
	"reflect"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	mon "github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AccountService provides business logic for account operations including creation, deposits, withdrawals, and balance inquiries.
type AccountService struct {
	uow       repository.UnitOfWork
	converter mon.CurrencyConverter
	logger    *slog.Logger
}

// NewAccountService creates a new AccountService with a UnitOfWork, CurrencyConverter, and logger.
func NewAccountService(
	uow repository.UnitOfWork,
	converter mon.CurrencyConverter,
	logger *slog.Logger,
) *AccountService {
	return &AccountService{
		uow:       uow,
		converter: converter,
		logger:    logger,
	}
}

// CreateAccount creates a new account for the specified user in a transaction.
func (s *AccountService) CreateAccount(ctx context.Context, userID uuid.UUID) (a *account.Account, err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem())
		if err != nil {
			return err
		}
		repo := repoAny.(repository.AccountRepository)
		a, err = account.New().WithUserID(userID).Build()
		if err != nil {
			return err
		}
		return repo.Create(a)
	})
	if err != nil {
		a = nil
	}
	return
}

// CreateAccountWithCurrency creates a new account for the specified user with a specific currency.
// This method allows creating accounts in different currencies, which is useful for multi-currency
// applications where users may need accounts in various currencies.
//
// The operation is wrapped with automatic transaction management and includes comprehensive
// error handling and logging.
//
// Parameters:
//   - userID: The UUID of the user who will own the account
//   - currencyCode: The ISO 4217 currency code for the account (e.g., "USD", "EUR", "JPY")
//
// Returns:
//   - A pointer to the created account with the specified currency
//   - An error if the operation fails (e.g., invalid currency, user not found, database error)
//
// The method validates the currency code and ensures it's supported by the system.
// If the currency is not supported, an appropriate domain error is returned.
//
// Example:
//
//	account, err := service.CreateAccountWithCurrency(userID, currency.Code("EUR"))
//	if err != nil {
//	    log.Error("Failed to create EUR account", "error", err)
//	    return
//	}
//	log.Info("EUR account created", "accountID", account.ID, "currency", account.Currency)
func (s *AccountService) CreateAccountWithCurrency(
	userID uuid.UUID,
	currencyCode currency.Code,
) (acct *account.Account, err error) {
	logger := s.logger.With("userID", userID, "currency", currencyCode)
	logger.Info("CreateAccountWithCurrency started")
	err = s.uow.Do(context.Background(), func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem())
		if err != nil {
			logger.Error("CreateAccountWithCurrency failed: AccountRepository error", "error", err)
			return err
		}
		repo := repoAny.(repository.AccountRepository)
		acct, err = account.New().
			WithUserID(userID).
			WithCurrency(currencyCode).
			Build()
		if err != nil {
			logger.Error("CreateAccountWithCurrency failed: domain error", "error", err)
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
// The method supports multi-currency deposits with automatic currency conversion
// when the deposit currency differs from the account currency.
//
// The operation is wrapped with automatic transaction management and includes
// comprehensive validation, error handling, and logging.
//
// Key Features:
// - Multi-currency support with real-time conversion
// - Automatic transaction record creation
// - Comprehensive validation (positive amounts, valid currencies)
// - User authorization checks
// - Detailed logging for observability
//
// Parameters:
//   - userID: The UUID of the user making the deposit (must own the account)
//   - accountID: The UUID of the account to deposit into
//   - amount: The amount to deposit (must be positive)
//   - currencyCode: The ISO 4217 currency code of the deposit amount
//
// Returns:
//   - A pointer to the created transaction record
//   - A pointer to conversion information (if currency conversion occurred)
//   - An error if the operation fails
//
// Currency Conversion:
// If the deposit currency differs from the account currency, the system will:
// 1. Fetch real-time exchange rates from the configured provider
// 2. Convert the amount to the account's currency
// 3. Store conversion details for audit purposes
// 4. Update the account balance with the converted amount
//
// Error Scenarios:
// - Account not found: Returns domain.ErrAccountNotFound
// - User not authorized: Returns domain.ErrUserUnauthorized
// - Invalid amount: Returns domain.ErrTransactionAmountMustBePositive
// - Invalid currency: Returns domain.ErrInvalidCurrencyCode
// - Insufficient funds: Returns domain.ErrInsufficientFunds
// - Conversion failure: Returns conversion service error
//
// Example:
//
//	tx, convInfo, err := service.Deposit(userID, accountID, 100.0, currency.Code("EUR"))
//	if err != nil {
//	    log.Error("Deposit failed", "error", err)
//	    return
//	}
//	if convInfo != nil {
//	    log.Info("Currency conversion applied",
//	        "originalAmount", convInfo.OriginalAmount,
//	        "convertedAmount", convInfo.ConvertedAmount,
//	        "rate", convInfo.ConversionRate)
//	}
func (s *AccountService) Deposit(
	userID, accountID uuid.UUID,
	amount float64,
	currencyCode currency.Code,
) (tx *account.Transaction, convInfo *common.ConversionInfo, err error) {
	s.logger.Info("Deposit started", "userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode)
	defer func() {
		if err != nil {
			s.logger.Error("Deposit failed", "userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode, "error", err)
		} else {
			s.logger.Info("Deposit successful", "userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode, "transactionID", tx.ID)
		}
	}()
	logger := s.logger.With("userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode)
	logger.Info("Deposit started")

	var txLocal *account.Transaction
	var convInfoLocal *common.ConversionInfo
	err = s.uow.Do(context.Background(), func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem())
		if err != nil {
			logger.Error("Deposit failed: AccountRepository error", "error", err)
			return err
		}
		repo := repoAny.(repository.AccountRepository)

		txRepoAny, err := uow.GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem())
		if err != nil {
			logger.Error("Deposit failed: TransactionRepository error", "error", err)
			return err
		}
		txRepo := txRepoAny.(repository.TransactionRepository)

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
			amountDeposit, err = mon.NewMoney(convInfoLocal.ConvertedAmount, a.Currency)
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
	s.logger.Info("Withdraw started", "userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode)
	defer func() {
		if err != nil {
			s.logger.Error("Withdraw failed", "userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode, "error", err)
		} else {
			s.logger.Info("Withdraw successful", "userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode, "transactionID", tx.ID)
		}
	}()
	logger := s.logger.With("userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode)
	logger.Info("Withdraw started")

	var txLocal *account.Transaction
	var convInfoLocal *common.ConversionInfo
	err = s.uow.Do(context.Background(), func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem())
		if err != nil {
			logger.Error("Withdraw failed: AccountRepository error", "error", err)
			return err
		}
		repo := repoAny.(repository.AccountRepository)

		txRepoAny, err := uow.GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem())
		if err != nil {
			logger.Error("Withdraw failed: TransactionRepository error", "error", err)
			return err
		}
		txRepo := txRepoAny.(repository.TransactionRepository)

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
			m, err = mon.NewMoney(convInfoLocal.ConvertedAmount, a.Currency)
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
	return
}

// GetAccount retrieves an account by its ID.
// Returns the account or an error if not found.
func (s *AccountService) GetAccount(
	userID, accountID uuid.UUID,
) (a *account.Account, err error) {
	s.logger.Info("GetAccount started", "userID", userID, "accountID", accountID)
	defer func() {
		if err != nil {
			s.logger.Error("GetAccount failed", "userID", userID, "accountID", accountID, "error", err)
		} else {
			s.logger.Info("GetAccount successful", "userID", userID, "accountID", accountID)
		}
	}()
	logger := s.logger.With("userID", userID, "accountID", accountID)
	logger.Info("GetAccount started")
	err = s.uow.Do(context.Background(), func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem())
		if err != nil {
			logger.Error("GetAccount failed: AccountRepository error", "error", err)
			return err
		}
		repo := repoAny.(repository.AccountRepository)
		aLocal, err := repo.Get(accountID)
		if err != nil {
			logger.Error("GetAccount failed: db error", "error", err)
			err = account.ErrAccountNotFound
			return err
		}
		if aLocal == nil {
			logger.Error("GetAccount failed: account not found", "error", err)
			err = account.ErrAccountNotFound
			return err

		}
		if aLocal.UserID != userID {
			logger.Error("GetAccount failed: user unauthorized", "accountUserID", aLocal.UserID)
			err = user.ErrUserUnauthorized
			return err
		}
		a = aLocal
		return nil
	})
	if err != nil {
		a = nil
	}
	return
}

// GetTransactions retrieves all transactions for a given account ID.
// Returns a slice of transactions or an error if the operation fails.
func (s *AccountService) GetTransactions(
	userID, accountID uuid.UUID,
) (txs []*account.Transaction, err error) {
	s.logger.Info("GetTransactions started", "userID", userID, "accountID", accountID)
	defer func() {
		if err != nil {
			s.logger.Error("GetTransactions failed", "userID", userID, "accountID", accountID, "error", err)
		} else {
			s.logger.Info("GetTransactions successful", "userID", userID, "accountID", accountID, "count", len(txs))
		}
	}()
	logger := s.logger.With("userID", userID, "accountID", accountID)
	logger.Info("GetTransactions started")
	err = s.uow.Do(context.Background(), func(uow repository.UnitOfWork) error {
		txRepoAny, err := uow.GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem())
		if err != nil {
			logger.Error("GetTransactions failed: TransactionRepository error", "error", err)
			return err
		}
		txRepo := txRepoAny.(repository.TransactionRepository)
		txs, err = txRepo.List(userID, accountID)
		if err != nil {
			logger.Error("GetTransactions failed: repo list error", "error", err)
			return err
		}
		return nil
	})
	if err != nil {
		txs = nil
	}
	return
}

// GetBalance retrieves the current balance of the specified account.
// Returns the balance as a float64 or an error if the operation fails.
func (s *AccountService) GetBalance(
	userID, accountID uuid.UUID,
) (balance float64, err error) {
	s.logger.Info("GetBalance started", "userID", userID, "accountID", accountID)
	defer func() {
		if err != nil {
			s.logger.Error("GetBalance failed", "userID", userID, "accountID", accountID, "error", err)
		} else {
			s.logger.Info("GetBalance successful", "userID", userID, "accountID", accountID, "balance", balance)
		}
	}()
	logger := s.logger.With("userID", userID, "accountID", accountID)
	logger.Info("GetBalance started")
	err = s.uow.Do(context.Background(), func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem())
		if err != nil {
			logger.Error("GetBalance failed: AccountRepository error", "error", err)
			return err
		}
		repo := repoAny.(repository.AccountRepository)
		a, err := repo.Get(accountID)
		if err != nil {
			logger.Error("GetBalance failed: AccountRepository.Get error", "error", err)
			// Map GORM "record not found" to domain error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return account.ErrAccountNotFound
			}
			return err
		}
		if a == nil {
			err = account.ErrAccountNotFound
			logger.Error("GetBalance failed:  ErrAccountNotFound")
			return err
		}
		balance, err = a.GetBalance(userID)
		if err != nil {
			logger.Error("GetBalance failed: domain error", "error", err)
			return err
		}
		return nil
	})
	if err != nil {
		balance = 0
	}
	return
}
