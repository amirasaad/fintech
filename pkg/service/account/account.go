// Package service provides business logic for interacting with domain entities such as accounts and transactions.
// It defines the AccountService struct and its methods for creating accounts, depositing and withdrawing funds,
// retrieving account details, listing transactions, and checking account balances.
//
// The service layer follows clean architecture principles and uses the decorator pattern for transaction management.
// All business operations are wrapped with automatic transaction management, error recovery, and structured logging.
package account

import (
	"context"
	"log/slog"
	"reflect"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	mon "github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

// OperationType represents the type of account operation
type OperationType string

const (
	OperationDeposit  OperationType = "deposit"
	OperationWithdraw OperationType = "withdraw"
)

// operationRequest contains the common parameters for account operations
type operationRequest struct {
	userID       uuid.UUID
	accountID    uuid.UUID
	amount       float64
	currencyCode currency.Code
	operation    OperationType
}

// operationResult contains the result of an account operation
type operationResult struct {
	transaction *account.Transaction
	convInfo    *common.ConversionInfo
}

// operationHandler defines the interface for handling account operations
type operationHandler interface {
	execute(account *account.Account, userID uuid.UUID, money mon.Money) (*account.Transaction, error)
}

// depositHandler implements operationHandler for deposit operations
type depositHandler struct{}

func (h *depositHandler) execute(account *account.Account, userID uuid.UUID, money mon.Money) (*account.Transaction, error) {
	return account.Deposit(userID, money)
}

// withdrawHandler implements operationHandler for withdraw operations
type withdrawHandler struct{}

func (h *withdrawHandler) execute(account *account.Account, userID uuid.UUID, money mon.Money) (*account.Transaction, error) {
	return account.Withdraw(userID, money)
}

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
	req := operationRequest{
		userID:       userID,
		accountID:    accountID,
		amount:       amount,
		currencyCode: currencyCode,
		operation:    OperationDeposit,
	}
	
	result, err := s.executeOperation(req, &depositHandler{})
	if err != nil {
		return nil, nil, err
	}
	
	return result.transaction, result.convInfo, nil
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
	req := operationRequest{
		userID:       userID,
		accountID:    accountID,
		amount:       amount,
		currencyCode: currencyCode,
		operation:    OperationWithdraw,
	}
	
	result, err := s.executeOperation(req, &withdrawHandler{})
	if err != nil {
		return nil, nil, err
	}
	
	return result.transaction, result.convInfo, nil
}

// executeOperation is the core method that handles both deposit and withdraw operations
// using the strategy pattern to eliminate code duplication and reduce branching.
func (s *AccountService) executeOperation(req operationRequest, handler operationHandler) (result *operationResult, err error) {
	logger := s.logger.With(
		"userID", req.userID,
		"accountID", req.accountID,
		"amount", req.amount,
		"currency", req.currencyCode,
		"operation", req.operation,
	)
	
	logger.Info("executeOperation started")
	defer func() {
		if err != nil {
			logger.Error("executeOperation failed", "error", err)
		} else {
			logger.Info("executeOperation successful", "transactionID", result.transaction.ID)
		}
	}()

	var txLocal *account.Transaction
	var convInfoLocal *common.ConversionInfo
	
	err = s.uow.Do(context.Background(), func(uow repository.UnitOfWork) error {
		// Get repositories
		accountRepo, txRepo, err := s.getRepositories(uow, logger)
		if err != nil {
			return err
		}

		// Get and validate account
		account, err := s.getAndValidateAccount(accountRepo, req.accountID, logger)
		if err != nil {
			return err
		}

		// Create money object
		money, err := s.createMoney(req.amount, req.currencyCode, logger)
		if err != nil {
			return err
		}

		// Handle currency conversion if needed
		convertedMoney, convInfo, err := s.handleCurrencyConversion(money, account.Currency, logger)
		if err != nil {
			return err
		}
		convInfoLocal = convInfo

		// Execute the operation using the strategy pattern
		txLocal, err = handler.execute(account, req.userID, convertedMoney)
		if err != nil {
			logger.Error("executeOperation failed: domain operation error", "error", err)
			return err
		}

		// Store conversion info if conversion occurred
		if convInfoLocal != nil {
			s.storeConversionInfo(txLocal, convInfoLocal, logger)
		}

		// Update account and create transaction
		if err = accountRepo.Update(account); err != nil {
			logger.Error("executeOperation failed: repo update error", "error", err)
			return err
		}
		if err = txRepo.Create(txLocal); err != nil {
			logger.Error("executeOperation failed: transaction create error", "error", err)
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &operationResult{
		transaction: txLocal,
		convInfo:    convInfoLocal,
	}, nil
}

// getRepositories retrieves the account and transaction repositories from the unit of work
func (s *AccountService) getRepositories(uow repository.UnitOfWork, logger *slog.Logger) (repository.AccountRepository, repository.TransactionRepository, error) {
	repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem())
	if err != nil {
		logger.Error("getRepositories failed: AccountRepository error", "error", err)
		return nil, nil, err
	}
	accountRepo := repoAny.(repository.AccountRepository)

	txRepoAny, err := uow.GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem())
	if err != nil {
		logger.Error("getRepositories failed: TransactionRepository error", "error", err)
		return nil, nil, err
	}
	txRepo := txRepoAny.(repository.TransactionRepository)

	return accountRepo, txRepo, nil
}

// getAndValidateAccount retrieves an account and validates it exists
func (s *AccountService) getAndValidateAccount(repo repository.AccountRepository, accountID uuid.UUID, logger *slog.Logger) (*account.Account, error) {
	account, err := repo.Get(accountID)
	if err != nil {
		logger.Error("getAndValidateAccount failed: account not found", "error", err)
		return nil, account.ErrAccountNotFound
	}
	return account, nil
}

// createMoney creates a Money object from amount and currency
func (s *AccountService) createMoney(amount float64, currencyCode currency.Code, logger *slog.Logger) (mon.Money, error) {
	money, err := mon.NewMoney(amount, currencyCode)
	if err != nil {
		logger.Error("createMoney failed: invalid money", "error", err)
		return mon.Money{}, err
	}
	return money, nil
}

// handleCurrencyConversion handles currency conversion if the money currency differs from account currency
func (s *AccountService) handleCurrencyConversion(money mon.Money, accountCurrency currency.Code, logger *slog.Logger) (mon.Money, *common.ConversionInfo, error) {
	if money.Currency() == accountCurrency {
		// No conversion needed
		return money, nil, nil
	}

	convInfo, err := s.converter.Convert(money.AmountFloat(), string(money.Currency()), string(accountCurrency))
	if err != nil {
		logger.Error("handleCurrencyConversion failed: currency conversion error", "error", err)
		return mon.Money{}, nil, err
	}

	convertedMoney, err := mon.NewMoney(convInfo.ConvertedAmount, accountCurrency)
	if err != nil {
		logger.Error("handleCurrencyConversion failed: converted money creation error", "error", err)
		return mon.Money{}, nil, err
	}

	return convertedMoney, convInfo, nil
}

// storeConversionInfo stores conversion information in the transaction
func (s *AccountService) storeConversionInfo(tx *account.Transaction, convInfo *common.ConversionInfo, logger *slog.Logger) {
	logger.Info("storeConversionInfo: conversion info stored",
		"originalAmount", convInfo.OriginalAmount,
		"originalCurrency", convInfo.OriginalCurrency,
		"conversionRate", convInfo.ConversionRate,
	)
	tx.OriginalAmount = &convInfo.OriginalAmount
	tx.OriginalCurrency = &convInfo.OriginalCurrency
	tx.ConversionRate = &convInfo.ConversionRate
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
			return err
		}
		if aLocal.UserID != userID {
			logger.Error("GetAccount failed: user unauthorized", "accountUserID", aLocal.UserID)
			return user.ErrUserUnauthorized
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
		// First, verify the account exists and belongs to the user
		repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem())
		if err != nil {
			logger.Error("GetTransactions failed: AccountRepository error", "error", err)
			return err
		}
		repo := repoAny.(repository.AccountRepository)
		a, err := repo.Get(accountID)
		if err != nil {
			logger.Error("GetTransactions failed: account not found", "error", err)
			return err
		}
		if a == nil {
			logger.Error("GetTransactions failed: account not found")
			return account.ErrAccountNotFound
		}
		if a.UserID != userID {
			logger.Error("GetTransactions failed: user unauthorized", "accountUserID", a.UserID)
			return user.ErrUserUnauthorized
		}

		// Now get the transactions
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
