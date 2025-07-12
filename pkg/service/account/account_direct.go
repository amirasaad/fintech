package account

import (
	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	mon "github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

// DirectAccountService uses direct repository injection instead of UOW with reflect
type DirectAccountService struct {
	accountRepo     repository.AccountRepository
	transactionRepo repository.TransactionRepository
	converter       mon.CurrencyConverter
	logger          *slog.Logger
}

// NewDirectAccountService creates a new service with direct repository injection
func NewDirectAccountService(
	accountRepo repository.AccountRepository,
	transactionRepo repository.TransactionRepository,
	converter mon.CurrencyConverter,
	logger *slog.Logger,
) *DirectAccountService {
	return &DirectAccountService{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
		converter:       converter,
		logger:          logger,
	}
}

// Deposit adds funds to the specified account without using reflect
func (s *DirectAccountService) Deposit(
	userID, accountID uuid.UUID,
	amount float64,
	currencyCode currency.Code,
) (tx *account.Transaction, convInfo *common.ConversionInfo, err error) {
	logger := s.logger.With("userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode)
	logger.Info("Deposit started")

	// Get account directly - no reflect needed
	acc, err := s.accountRepo.Get(accountID)
	if err != nil {
		logger.Error("Deposit failed: account not found", "error", err)
		return nil, nil, account.ErrAccountNotFound
	}

	// Create money
	money, err := mon.NewMoney(amount, currencyCode)
	if err != nil {
		logger.Error("Deposit failed: invalid money", "error", err)
		return nil, nil, err
	}

	// Handle currency conversion
	convertedMoney, convInfo, err := s.handleCurrencyConversion(money, acc.Currency, logger)
	if err != nil {
		return nil, nil, err
	}

	// Execute deposit
	tx, err = acc.Deposit(userID, convertedMoney)
	if err != nil {
		logger.Error("Deposit failed: domain deposit error", "error", err)
		return nil, nil, err
	}

	// Store conversion info
	if convInfo != nil {
		tx.OriginalAmount = &convInfo.OriginalAmount
		tx.OriginalCurrency = &convInfo.OriginalCurrency
		tx.ConversionRate = &convInfo.ConversionRate
	}

	// Update account and create transaction
	if err = s.accountRepo.Update(acc); err != nil {
		logger.Error("Deposit failed: repo update error", "error", err)
		return nil, nil, err
	}
	if err = s.transactionRepo.Create(tx); err != nil {
		logger.Error("Deposit failed: transaction create error", "error", err)
		return nil, nil, err
	}

	logger.Info("Deposit successful", "transactionID", tx.ID)
	return tx, convInfo, nil
}

// Withdraw removes funds from the specified account without using reflect
func (s *DirectAccountService) Withdraw(
	userID, accountID uuid.UUID,
	amount float64,
	currencyCode currency.Code,
) (tx *account.Transaction, convInfo *common.ConversionInfo, err error) {
	logger := s.logger.With("userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode)
	logger.Info("Withdraw started")

	// Get account directly - no reflect needed
	acc, err := s.accountRepo.Get(accountID)
	if err != nil {
		logger.Error("Withdraw failed: account not found", "error", err)
		return nil, nil, account.ErrAccountNotFound
	}

	// Create money
	money, err := mon.NewMoney(amount, currencyCode)
	if err != nil {
		logger.Error("Withdraw failed: invalid money", "error", err)
		return nil, nil, err
	}

	// Handle currency conversion
	convertedMoney, convInfo, err := s.handleCurrencyConversion(money, acc.Currency, logger)
	if err != nil {
		return nil, nil, err
	}

	// Execute withdraw
	tx, err = acc.Withdraw(userID, convertedMoney)
	if err != nil {
		logger.Error("Withdraw failed: domain withdraw error", "error", err)
		return nil, nil, err
	}

	// Store conversion info
	if convInfo != nil {
		tx.OriginalAmount = &convInfo.OriginalAmount
		tx.OriginalCurrency = &convInfo.OriginalCurrency
		tx.ConversionRate = &convInfo.ConversionRate
	}

	// Update account and create transaction
	if err = s.accountRepo.Update(acc); err != nil {
		logger.Error("Withdraw failed: repo update error", "error", err)
		return nil, nil, err
	}
	if err = s.transactionRepo.Create(tx); err != nil {
		logger.Error("Withdraw failed: transaction create error", "error", err)
		return nil, nil, err
	}

	logger.Info("Withdraw successful", "transactionID", tx.ID)
	return tx, convInfo, nil
}

// handleCurrencyConversion handles currency conversion if needed
func (s *DirectAccountService) handleCurrencyConversion(
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
