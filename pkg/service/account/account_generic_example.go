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

// GenericAccountService uses generic repositories for type-safe data access
type GenericAccountService struct {
	accountRepo     repository.GenericRepository[account.Account]
	transactionRepo repository.GenericRepository[account.Transaction]
	converter       mon.CurrencyConverter
	logger          *slog.Logger
}

// NewGenericAccountService creates a new service with generic repositories
func NewGenericAccountService(
	accountRepo repository.GenericRepository[account.Account],
	transactionRepo repository.GenericRepository[account.Transaction],
	converter mon.CurrencyConverter,
	logger *slog.Logger,
) *GenericAccountService {
	return &GenericAccountService{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
		converter:       converter,
		logger:          logger,
	}
}

// Deposit adds funds to the specified account using generic repositories
func (s *GenericAccountService) Deposit(
	userID, accountID uuid.UUID,
	amount float64,
	currencyCode currency.Code,
) (tx *account.Transaction, convInfo *common.ConversionInfo, err error) {
	logger := s.logger.With("userID", userID, "accountID", accountID, "amount", amount, "currency", currencyCode)
	logger.Info("Deposit started")

	// Get account using generic repository - type-safe!
	acc, err := s.accountRepo.Get(context.Background(), accountID)
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

	// Update account and create transaction using generic repositories
	if err = s.accountRepo.Update(context.Background(), acc); err != nil {
		logger.Error("Deposit failed: repo update error", "error", err)
		return nil, nil, err
	}
	if err = s.transactionRepo.Create(context.Background(), tx); err != nil {
		logger.Error("Deposit failed: transaction create error", "error", err)
		return nil, nil, err
	}

	logger.Info("Deposit successful", "transactionID", tx.ID)
	return tx, convInfo, nil
}

// GetAccountTransactions retrieves transactions for an account using generic repository
func (s *GenericAccountService) GetAccountTransactions(accountID uuid.UUID) ([]*account.Transaction, error) {
	// Use generic repository's FindBy method
	transactions, err := s.transactionRepo.FindBy(context.Background(), "account_id = ?", accountID)
	if err != nil {
		return nil, err
	}
	return transactions, nil
}

// GetAccountsByUser retrieves all accounts for a user using generic repository
func (s *GenericAccountService) GetAccountsByUser(userID uuid.UUID) ([]*account.Account, error) {
	// Use generic repository's FindBy method
	accounts, err := s.accountRepo.FindBy(context.Background(), "user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

// handleCurrencyConversion handles currency conversion if needed
func (s *GenericAccountService) handleCurrencyConversion(
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
