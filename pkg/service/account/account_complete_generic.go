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

// CompleteGenericAccountService uses the complete generic UOW pattern
type CompleteGenericAccountService struct {
	uow       repository.CompleteGenericUnitOfWork
	converter mon.CurrencyConverter
	logger    *slog.Logger
}

// NewCompleteGenericAccountService creates a new service with complete generic UOW
func NewCompleteGenericAccountService(
	uow repository.CompleteGenericUnitOfWork,
	converter mon.CurrencyConverter,
	logger *slog.Logger,
) *CompleteGenericAccountService {
	return &CompleteGenericAccountService{
		uow:       uow,
		converter: converter,
		logger:    logger,
	}
}

// Deposit adds funds to the specified account using complete generic UOW
func (s *CompleteGenericAccountService) Deposit(
	userID, accountID uuid.UUID,
	amount float64,
	currencyCode currency.Code,
) (tx *account.Transaction, convInfo *common.ConversionInfo, err error) {
	logger := s.logger.With("userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode)
	logger.Info("Deposit started")

	err = s.uow.Do(context.Background(), func(uow repository.CompleteGenericUnitOfWork) error {
		// Type-safe repository access using generics - no reflect needed!
		accountRepo := uow.AccountRepository()
		transactionRepo := uow.TransactionRepository()

		// Get account using generic repository
		acc, err := accountRepo.Get(context.Background(), accountID)
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

		// Update account and create transaction using generic repositories
		if err = accountRepo.Update(context.Background(), acc); err != nil {
			logger.Error("Deposit failed: repo update error", "error", err)
			return err
		}
		if err = transactionRepo.Create(context.Background(), tx); err != nil {
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

// GetAccountTransactions retrieves transactions for an account using generic repository
func (s *CompleteGenericAccountService) GetAccountTransactions(accountID uuid.UUID) ([]*account.Transaction, error) {
	var transactions []*account.Transaction

	err := s.uow.Do(context.Background(), func(uow repository.CompleteGenericUnitOfWork) error {
		transactionRepo := uow.TransactionRepository()

		// Use generic repository's FindBy method
		txs, err := transactionRepo.FindBy(context.Background(), "account_id = ?", accountID)
		if err != nil {
			return err
		}
		transactions = txs
		return nil
	})

	return transactions, err
}

// GetAccountsByUser retrieves all accounts for a user using generic repository
func (s *CompleteGenericAccountService) GetAccountsByUser(userID uuid.UUID) ([]*account.Account, error) {
	var accounts []*account.Account

	err := s.uow.Do(context.Background(), func(uow repository.CompleteGenericUnitOfWork) error {
		accountRepo := uow.AccountRepository()

		// Use generic repository's FindBy method
		accs, err := accountRepo.FindBy(context.Background(), "user_id = ?", userID)
		if err != nil {
			return err
		}
		accounts = accs
		return nil
	})

	return accounts, err
}

// handleCurrencyConversion handles currency conversion if needed
func (s *CompleteGenericAccountService) handleCurrencyConversion(
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
