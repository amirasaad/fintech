// Package account provides business logic for interacting with domain entities such as accounts and transactions.
// It defines the Service struct and its methods for creating accounts, depositing and withdrawing funds,
// retrieving account details, listing transactions, and checking account balances.
//
// The service layer follows clean architecture principles and uses the decorator pattern for transaction management.
// All business operations are wrapped with automatic transaction management, error recovery, and structured logging.
package account

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

// Service provides business logic for account operations including creation, deposits, withdrawals, and balance inquiries.
type Service struct {
	uow           repository.UnitOfWork
	converter     money.CurrencyConverter
	logger        *slog.Logger
	depositChain  OperationHandler
	withdrawChain OperationHandler
	transferChain OperationHandler
}

// NewAccountService creates a new Service with a UnitOfWork, CurrencyConverter, and logger.
func NewAccountService(
	uow repository.UnitOfWork,
	converter money.CurrencyConverter,
	logger *slog.Logger,
) *Service {
	builder := NewChainBuilder(uow, converter, logger)
	return &Service{
		uow:           uow,
		converter:     converter,
		logger:        logger,
		depositChain:  builder.BuildDepositChain(),
		withdrawChain: builder.BuildWithdrawChain(),
		transferChain: builder.BuildTransferChain(),
	}
}

// CreateAccount creates a new account for the specified user in a transaction.
func (s *Service) CreateAccount(ctx context.Context, userID uuid.UUID) (a *account.Account, err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repo, err := uow.AccountRepository()
		if err != nil {
			return err
		}
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
func (s *Service) CreateAccountWithCurrency(
	userID uuid.UUID,
	currencyCode currency.Code,
) (acct *account.Account, err error) {
	logger := s.logger.With("userID", userID, "currency", currencyCode)
	logger.Info("CreateAccountWithCurrency started")
	err = s.uow.Do(context.Background(), func(uow repository.UnitOfWork) error {
		repo, err := uow.AccountRepository()
		if err != nil {
			logger.Error("CreateAccountWithCurrency failed: AccountRepository error", "error", err)
			return err
		}
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
func (s *Service) Deposit(
	userID, accountID uuid.UUID,
	amount float64,
	currencyCode currency.Code,
) (tx *account.Transaction, convInfo *common.ConversionInfo, err error) {
	m, err := money.New(amount, currencyCode)
	if err != nil {
		return nil, nil, err
	}
	s.logger.Info("Deposit: starting", "userID", userID, "accountID", accountID, "amount", m.Amount(), "currency", currencyCode, "chain_nil", s.depositChain == nil)

	req := &OperationRequest{
		UserID:       userID,
		AccountID:    accountID,
		Amount:       amount,
		CurrencyCode: currencyCode,
		Operation:    OperationDeposit,
	}

	resp, err := s.depositChain.Handle(context.Background(), req)
	if err != nil {
		return nil, nil, err
	}

	if resp.Error != nil {
		return nil, nil, resp.Error
	}

	return resp.Transaction, resp.ConvInfo, nil
}

// Withdraw removes funds from the specified account and creates a transaction record.
// The method supports multi-currency withdrawals with automatic currency conversion
// when the withdrawal currency differs from the account currency.
//
// The operation is wrapped with automatic transaction management and includes
// comprehensive validation, error handling, and logging.
//
// Key Features:
// - Multi-currency support with real-time conversion
// - Automatic transaction record creation
// - Comprehensive validation (positive amounts, valid currencies)
// - User authorization checks
// - Insufficient funds validation
// - Detailed logging for observability
//
// Parameters:
//   - userID: The UUID of the user making the withdrawal (must own the account)
//   - accountID: The UUID of the account to withdraw from
//   - amount: The amount to withdraw (must be positive)
//   - currencyCode: The ISO 4217 currency code of the withdrawal amount
//
// Returns:
//   - A pointer to the created transaction record
//   - A pointer to conversion information (if currency conversion occurred)
//   - An error if the operation fails
//
// Currency Conversion:
// If the withdrawal currency differs from the account currency, the system will:
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
//	tx, convInfo, err := service.Withdraw(userID, accountID, 50.0, currency.Code("USD"))
//	if err != nil {
//	    log.Error("Withdraw failed", "error", err)
//	    return
//	}
//	if convInfo != nil {
//	    log.Info("Currency conversion applied",
//	        "originalAmount", convInfo.OriginalAmount,
//	        "convertedAmount", convInfo.ConvertedAmount,
//	        "rate", convInfo.ConversionRate)
//	}
func (s *Service) Withdraw(
	userID, accountID uuid.UUID,
	amount float64,
	currencyCode currency.Code,
) (
	tx *account.Transaction,
	convInfo *common.ConversionInfo,
	err error,
) {
	req := &OperationRequest{
		UserID:       userID,
		AccountID:    accountID,
		Amount:       amount,
		CurrencyCode: currencyCode,
		Operation:    OperationWithdraw,
	}

	resp, err := s.withdrawChain.Handle(context.Background(), req)
	if err != nil {
		return nil, nil, err
	}

	if resp.Error != nil {
		return nil, nil, resp.Error
	}

	return resp.Transaction, resp.ConvInfo, nil
}

// Transfer moves funds from one account to another account.
func (s *Service) Transfer(
	userID uuid.UUID,
	sourceAccountID, destAccountID uuid.UUID,
	amount float64,
	currencyCode currency.Code,
	moneySource account.MoneySource,
) (
	txOut, txIn *account.Transaction,
	err error,
) {
	m, err := money.New(amount, currencyCode)
	if err != nil {
		return
	}
	s.logger.Info("Transfer: starting", "userID", userID, "sourceAccountID", sourceAccountID, "destAccountID", destAccountID, "amount", m.Amount(), "currency", currencyCode, "moneySource", moneySource)
	req := &OperationRequest{
		UserID:        userID,
		AccountID:     sourceAccountID,
		Amount:        amount,
		CurrencyCode:  currencyCode,
		Operation:     OperationTransfer,
		DestAccountID: destAccountID,
		DestUserID:    userID, // TODO: fetch actual dest user if needed
	}
	resp, err := s.transferChain.Handle(context.Background(), req)
	if err != nil {
		s.logger.Error("Transfer: chain failed", "error", err)
		return
	}
	if resp.Error != nil {
		s.logger.Error("Transfer: business error", "error", resp.Error)
		err = resp.Error
		return
	}
	txOut = resp.TransactionOut
	txIn = resp.TransactionIn
	return txOut, txIn, nil
}
