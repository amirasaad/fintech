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
	"github.com/amirasaad/fintech/pkg/repository"
	mon "github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
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
