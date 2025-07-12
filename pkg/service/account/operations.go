package account

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	mon "github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

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
	// Use type-safe convenience methods instead of reflect
	accountRepo, err := uow.AccountRepository()
	if err != nil {
		logger.Error("getRepositories failed: AccountRepository error", "error", err)
		return nil, nil, err
	}

	txRepo, err := uow.TransactionRepository()
	if err != nil {
		logger.Error("getRepositories failed: TransactionRepository error", "error", err)
		return nil, nil, err
	}

	return accountRepo, txRepo, nil
}

// getAndValidateAccount retrieves an account and validates it exists
func (s *AccountService) getAndValidateAccount(repo repository.AccountRepository, accountID uuid.UUID, logger *slog.Logger) (*account.Account, error) {
	acc, err := repo.Get(accountID)
	if err != nil {
		logger.Error("getAndValidateAccount failed: account not found", "error", err)
		return nil, account.ErrAccountNotFound
	}
	return acc, nil
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
