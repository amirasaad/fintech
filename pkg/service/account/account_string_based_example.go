package account

import (
	"context"
	"log/slog"

	infraRepo "github.com/amirasaad/fintech/infra/repository"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	mon "github.com/amirasaad/fintech/pkg/domain/money"
	repo "github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

// StringBasedAccountService demonstrates the string-based UOW approach
type StringBasedAccountService struct {
	uow       infraRepo.StringBasedUnitOfWork
	converter mon.CurrencyConverter
	logger    *slog.Logger
}

// NewStringBasedAccountService creates a new service with string-based UOW
func NewStringBasedAccountService(
	uow infraRepo.StringBasedUnitOfWork,
	converter mon.CurrencyConverter,
	logger *slog.Logger,
) *StringBasedAccountService {
	return &StringBasedAccountService{
		uow:       uow,
		converter: converter,
		logger:    logger,
	}
}

// Deposit demonstrates string-based repository access
func (s *StringBasedAccountService) Deposit(
	userID, accountID uuid.UUID,
	amount float64,
	currencyCode currency.Code,
) (tx *account.Transaction, convInfo *common.ConversionInfo, err error) {
	logger := s.logger.With("userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode)
	logger.Info("Deposit started")

	var txLocal *account.Transaction
	var convInfoLocal *common.ConversionInfo

	err = s.uow.Do(context.Background(), func(uow infraRepo.StringBasedUnitOfWork) error {
		// Option 1: Use string-based access (your suggested approach)
		accountRepoAny, err := uow.GetRepository("account")
		if err != nil {
			logger.Error("Deposit failed: AccountRepository error", "error", err)
			return err
		}
		accountRepo := accountRepoAny.(repo.AccountRepository)

		// Option 2: Use type-safe convenience methods (recommended)
		// accountRepo, err := uow.AccountRepository()
		// if err != nil {
		//     logger.Error("Deposit failed: AccountRepository error", "error", err)
		//     return err
		// }

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
		convInfoLocal = convInfo

		// Execute deposit
		txLocal, err = acc.Deposit(userID, convertedMoney)
		if err != nil {
			logger.Error("Deposit failed: domain deposit error", "error", err)
			return err
		}

		// Store conversion info
		if convInfoLocal != nil {
			txLocal.OriginalAmount = &convInfoLocal.OriginalAmount
			txLocal.OriginalCurrency = &convInfoLocal.OriginalCurrency
			txLocal.ConversionRate = &convInfoLocal.ConversionRate
		}

		// Update account
		if err = accountRepo.Update(acc); err != nil {
			logger.Error("Deposit failed: repo update error", "error", err)
			return err
		}

		// Get transaction repository using string
		txRepoAny, err := uow.GetRepository("transaction")
		if err != nil {
			logger.Error("Deposit failed: TransactionRepository error", "error", err)
			return err
		}
		txRepo := txRepoAny.(repo.TransactionRepository)

		// Create transaction
		if err = txRepo.Create(txLocal); err != nil {
			logger.Error("Deposit failed: transaction create error", "error", err)
			return err
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	logger.Info("Deposit successful", "transactionID", txLocal.ID)
	return txLocal, convInfoLocal, nil
}

// handleCurrencyConversion handles currency conversion if needed
func (s *StringBasedAccountService) handleCurrencyConversion(
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

	logger.Info("Currency conversion applied",
		"original", money,
		"converted", convertedMoney,
		"rate", convInfo.ConversionRate)

	return convertedMoney, convInfo, nil
}
