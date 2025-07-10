// Package service provides business logic for interacting with domain entities such as accounts and transactions.
// It defines the AccountService struct and its methods for creating accounts, depositing and withdrawing funds,
// retrieving account details, listing transactions, and checking account balances.
//
// The service layer follows clean architecture principles and uses the decorator pattern for transaction management.
// All business operations are wrapped with automatic transaction handling, error recovery, and structured logging.
package service

import (
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	mon "github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/repository"

	"log/slog"

	"github.com/amirasaad/fintech/pkg/decorator"
	"github.com/google/uuid"
)

// AccountService provides business logic for account operations including creation, deposits, withdrawals,
// and balance inquiries. It implements the application layer of clean architecture and coordinates between
// domain entities and repositories.
//
// The service uses dependency injection for all dependencies and the decorator pattern for transaction
// management. All operations are automatically wrapped with transaction handling, error recovery, and
// structured logging.
//
// Key Features:
// - Automatic transaction management using the decorator pattern
// - Multi-currency support with real-time conversion
// - Comprehensive error handling and logging
// - Clean separation of business logic from infrastructure concerns
// - Thread-safe operations with proper concurrency handling
//
// Example usage:
//
//	uowFactory := func() (repository.UnitOfWork, error) {
//	    return infra.NewUnitOfWork(db)
//	}
//	converter := currency.NewConverter(apiKey)
//	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
//
//	accountService := NewAccountService(uowFactory, converter, logger)
//
//	account, err := accountService.CreateAccount(userID)
//	if err != nil {
//	    // Handle error - transaction was automatically rolled back
//	}
type AccountService struct {
	uowFactory  func() (repository.UnitOfWork, error)
	converter   mon.CurrencyConverter
	logger      *slog.Logger
	transaction decorator.TransactionDecorator
}

// NewAccountService creates a new AccountService instance with all required dependencies.
//
// The service is configured with:
// - Unit of Work factory for transaction management
// - Currency converter for multi-currency operations
// - Structured logger for observability
// - Transaction decorator for automatic transaction handling
//
// Parameters:
//   - uowFactory: A function that creates and returns a UnitOfWork instance. This factory
//     is used by the transaction decorator to manage transaction lifecycles.
//   - converter: A currency converter that handles real-time exchange rate conversion
//     for multi-currency operations. Can be nil for single-currency applications.
//   - logger: A structured logger for recording business operations, errors, and
//     debugging information. All operations are logged with appropriate context.
//
// Returns a fully configured AccountService ready for business operations.
//
// Example:
//
//	uowFactory := func() (repository.UnitOfWork, error) {
//	    return infra.NewUnitOfWork(db)
//	}
//	converter := currency.NewConverter(apiKey)
//	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
//
//	service := NewAccountService(uowFactory, converter, logger)
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

// CreateAccount creates a new account for the specified user and persists it using the repository.
// The account is created with default currency settings and automatically assigned a unique ID.
//
// The operation is wrapped with automatic transaction management using the decorator pattern.
// If any part of the operation fails, the transaction is automatically rolled back and an
// appropriate error is returned.
//
// Parameters:
//   - userID: The UUID of the user who will own the account. The user must exist in the system.
//
// Returns:
//   - A pointer to the created account with all fields populated
//   - An error if the operation fails (e.g., user not found, database error)
//
// The method includes comprehensive logging for observability:
// - Start of operation with user ID
// - Success with account ID
// - Failure with error details
//
// Example:
//
//	account, err := service.CreateAccount(userID)
//	if err != nil {
//	    log.Error("Failed to create account", "error", err)
//	    return
//	}
//	log.Info("Account created", "accountID", account.ID, "userID", account.UserID)
func (s *AccountService) CreateAccount(
	userID uuid.UUID,
) (a *account.Account, err error) {
	s.logger.Info("CreateAccount started", "userID", userID)
	defer func() {
		if err != nil {
			s.logger.Error("CreateAccount failed", "userID", userID, "error", err)
		} else {
			s.logger.Info("CreateAccount successful", "userID", userID, "accountID", a.ID)
		}
	}()
	var aLocal *account.Account
	err = s.transaction.Execute(func() error {
		aLocal, err = account.New().WithUserID(userID).Build()
		if err != nil {
			return err
		}
		uow, err := s.uowFactory()
		if err != nil {
			s.logger.Error("CreateAccount failed: uowFactory error", "userID", userID, "error", err)
			return err
		}
		repo, err := uow.AccountRepository()
		if err != nil {
			s.logger.Error("CreateAccount failed: AccountRepository error", "userID", userID, "error", err)
			return err
		}
		if err = repo.Create(aLocal); err != nil {
			s.logger.Error("CreateAccount failed: repo create error", "userID", userID, "error", err)
			return err
		}
		return nil
	})
	if err != nil {
		s.logger.Error("CreateAccount failed: transaction error", "userID", userID, "error", err)
		return nil, err
	}
	a = aLocal
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
	return
}
