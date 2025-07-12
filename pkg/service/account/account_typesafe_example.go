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

// TypeSafeAccountService uses the improved UOW with type-safe repository access
type TypeSafeAccountService struct {
	uow       repository.ImprovedUnitOfWork
	converter mon.CurrencyConverter
	logger    *slog.Logger
}

// NewTypeSafeAccountService creates a new service with type-safe UOW
func NewTypeSafeAccountService(
	uow repository.ImprovedUnitOfWork,
	converter mon.CurrencyConverter,
	logger *slog.Logger,
) *TypeSafeAccountService {
	return &TypeSafeAccountService{
		uow:       uow,
		converter: converter,
		logger:    logger,
	}
}

// Deposit adds funds to the specified account using type-safe UOW
func (s *TypeSafeAccountService) Deposit(
	userID, accountID uuid.UUID,
	amount float64,
	currencyCode currency.Code,
) (tx *account.Transaction, convInfo *common.ConversionInfo, err error) {
	logger := s.logger.With("userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode)
	logger.Info("Deposit started")

	err = s.uow.Do(context.Background(), func(uow repository.ImprovedUnitOfWork) error {
		// Type-safe repository access - no reflect needed!
		accountRepo := uow.AccountRepository()
		transactionRepo := uow.TransactionRepository()

		// Get account
		acc, err := accountRepo.Get(accountID)
		if err != nil {
			logger.Error("Deposit failed: account not found", "error", err)
			return account.ErrAccountNotFound
		}

		// Create money
		money, err := mon.NewMoney(amount, currencyCode)
		if err != nil {
			logger.Error("Deposit failed: invalid money", "error", err)
			return err
		}

		// Handle currency conversion
		convertedMoney, convInfo, err := s.handleCurrencyConversion(money, acc.Currency, logger)
		if err != nil {
			return err
		}

		// Execute deposit
		tx, err = acc.Deposit(userID, convertedMoney)
		if err != nil {
			logger.Error("Deposit failed: domain deposit error", "error", err)
			return err
		}

		// Store conversion info
		if convInfo != nil {
			tx.OriginalAmount = &convInfo.OriginalAmount
			tx.OriginalCurrency = &convInfo.OriginalCurrency
			tx.ConversionRate = &convInfo.ConversionRate
		}

		// Update account and create transaction
		if err = accountRepo.Update(acc); err != nil {
			logger.Error("Deposit failed: repo update error", "error", err)
			return err
		}
		if err = transactionRepo.Create(tx); err != nil {
			logger.Error("Deposit failed: transaction create error", "error", err)
			return err
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	logger.Info("Deposit successful", "transactionID", tx.ID)
	return tx, convInfo, nil
}

// handleCurrencyConversion handles currency conversion if needed
func (s *TypeSafeAccountService) handleCurrencyConversion(
	money mon.Money,
	accountCurrency currency.Code,
	logger *slog.Logger,
) (mon.Money, *common.ConversionInfo, error) {
	if money.Currency() == accountCurrency {
		return money, nil, nil
	}

	convInfo, err := s.converter.Convert(money.AmountFloat(), string(money.Currency()), string(accountCurrency))
	if err != nil {
		logger.Error("Currency conversion failed", "error", err)
		return mon.Money{}, nil, err
	}

	convertedMoney, err := mon.NewMoney(convInfo.ConvertedAmount, accountCurrency)
	if err != nil {
		logger.Error("Converted money creation failed", "error", err)
		return mon.Money{}, nil, err
	}

	return convertedMoney, convInfo, nil
}
