// Package service provides business logic for interacting with domain entities such as accounts and transactions.
// It defines the AccountService struct and its methods for creating accounts, depositing and withdrawing funds,
// retrieving account details, listing transactions, and checking account balances.
package service

import (
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"

	"github.com/google/uuid"
)

// AccountService provides methods to interact with accounts and transactions using a unit of work pattern.
type AccountService struct {
	uowFactory func() (repository.UnitOfWork, error)
	converter  domain.CurrencyConverter
}

// NewAccountService creates a new AccountService with the given UnitOfWork factory and CurrencyConverter.
func NewAccountService(uowFactory func() (repository.UnitOfWork, error), converter domain.CurrencyConverter) *AccountService {
	return &AccountService{
		uowFactory: uowFactory,
		converter:  converter,
	}
}

// CreateAccount creates a new account and persists it using the repository.
// Returns the created account or an error if the operation fails.
func (s *AccountService) CreateAccount(
	userID uuid.UUID,
) (a *domain.Account, err error) {
	uow, err := s.uowFactory()
	if err != nil {
		a = nil
		return
	}
	err = uow.Begin()
	if err != nil {
		a = nil
		return
	}

	a = domain.NewAccount(userID)
	repo, err := uow.AccountRepository()
	if err != nil {
		_ = uow.Rollback()
		a = nil
		return
	}
	err = repo.Create(a)
	if err != nil {
		_ = uow.Rollback()
		a = nil
		return
	}

	err = uow.Commit()
	if err != nil {
		_ = uow.Rollback()
		a = nil
		return
	}
	return
}

func (s *AccountService) CreateAccountWithCurrency(
	userID uuid.UUID,
	currency string) (account *domain.Account, err error) {
	uow, err := s.uowFactory()
	if err != nil {
		account = nil
		return
	}
	err = uow.Begin()
	if err != nil {
		account = nil
		return
	}

	account = domain.NewAccountWithCurrency(userID, currency)
	repo, err := uow.AccountRepository()
	if err != nil {
		_ = uow.Rollback()
		account = nil
		return
	}
	err = repo.Create(account)
	if err != nil {
		_ = uow.Rollback()
		account = nil
		return
	}

	err = uow.Commit()
	if err != nil {
		_ = uow.Rollback()
		account = nil
		return
	}
	return
}

// Deposit adds funds to the specified account and creates a transaction record.
// Returns the transaction or an error if the operation fails.
func (s *AccountService) Deposit(
	userID, accountID uuid.UUID,
	amount float64,
	currency string,
) (tx *domain.Transaction, convInfo *domain.ConversionInfo, err error) {
	money, err := domain.NewMoney(amount, currency)
	if err != nil {
		tx = nil
		return
	}
	uow, err := s.uowFactory()
	if err != nil {
		tx = nil
		return
	}
	err = uow.Begin()
	if err != nil {
		tx = nil
		return
	}

	repo, err := uow.AccountRepository()
	if err != nil {
		_ = uow.Rollback()
		tx = nil
		return
	}
	a, err := repo.Get(accountID)
	if err != nil {
		_ = uow.Rollback()
		tx = nil
		err = domain.ErrAccountNotFound
		return
	}

	if money.Currency != a.Currency {
		convInfo, err = s.converter.Convert(money.Amount, money.Currency, a.Currency)
		if err != nil {
			_ = uow.Rollback()
			tx = nil
			return
		}
		money, err = domain.NewMoney(convInfo.ConvertedAmount, a.Currency)
		if err != nil {
			_ = uow.Rollback()
			tx = nil
			return
		}
	}

	tx, err = a.Deposit(userID, money)
	if err != nil {
		_ = uow.Rollback()
		tx = nil
		return
	}

	// Store conversion info in transaction if conversion occurred
	if convInfo != nil {
		tx.OriginalAmount = &convInfo.OriginalAmount
		tx.OriginalCurrency = &convInfo.OriginalCurrency
		tx.ConversionRate = &convInfo.ConversionRate
	}

	err = repo.Update(a)
	if err != nil {
		_ = uow.Rollback()
		tx = nil
		return
	}

	txRepo, err := uow.TransactionRepository()
	if err != nil {
		_ = uow.Rollback()
		tx = nil
		return
	}
	err = txRepo.Create(tx)
	if err != nil {
		_ = uow.Rollback()
		tx = nil
		return
	}

	err = uow.Commit()
	if err != nil {
		_ = uow.Rollback()
		tx = nil
		return
	}

	return
}

// Withdraw removes funds from the specified account and creates a transaction record.
// Returns the transaction, conversion info (if any), and error if the operation fails.
func (s *AccountService) Withdraw(
	userID, accountID uuid.UUID,
	amount float64,
	currency string,
) (
	tx *domain.Transaction,
	convInfo *domain.ConversionInfo,
	err error,
) {
	money, err := domain.NewMoney(amount, currency)
	if err != nil {
		return nil, nil, err
	}
	uow, err := s.uowFactory()
	if err != nil {
		return nil, nil, err
	}
	err = uow.Begin()
	if err != nil {
		return nil, nil, err
	}

	repo, err := uow.AccountRepository()
	if err != nil {
		_ = uow.Rollback()
		return nil, nil, err
	}
	a, err := repo.Get(accountID)
	if err != nil {
		_ = uow.Rollback()
		return nil, nil, domain.ErrAccountNotFound
	}

	if money.Currency != a.Currency {
		convInfo, err = s.converter.Convert(money.Amount, money.Currency, a.Currency)
		if err != nil {
			_ = uow.Rollback()
			return
		}

		money, err = domain.NewMoney(convInfo.ConvertedAmount, a.Currency)
		if err != nil {
			_ = uow.Rollback()
			return nil, nil, err
		}
	}

	tx, err = a.Withdraw(userID, money)
	if err != nil {
		_ = uow.Rollback()
		return nil, nil, err
	}

	// Store conversion info in transaction if conversion occurred
	if convInfo != nil {
		tx.OriginalAmount = &convInfo.OriginalAmount
		tx.OriginalCurrency = &convInfo.OriginalCurrency
		tx.ConversionRate = &convInfo.ConversionRate
	}

	err = repo.Update(a)
	if err != nil {
		_ = uow.Rollback()
		return nil, nil, err
	}

	txRepo, err := uow.TransactionRepository()
	if err != nil {
		_ = uow.Rollback()
		return nil, nil, err
	}
	err = txRepo.Create(tx)
	if err != nil {
		_ = uow.Rollback()
		return nil, nil, err
	}

	err = uow.Commit()
	if err != nil {
		_ = uow.Rollback()
		return nil, nil, err
	}

	return tx, convInfo, nil
}

// GetAccount retrieves an account by its ID.
// Returns the account or an error if not found.
func (s *AccountService) GetAccount(
	userID, accountID uuid.UUID,
) (a *domain.Account, err error) {
	uow, err := s.uowFactory()
	if err != nil {
		a = nil
		return
	}

	repo, err := uow.AccountRepository()
	if err != nil {
		a = nil
		return
	}
	a, err = repo.Get(accountID)
	if err != nil {
		a = nil
		err = domain.ErrAccountNotFound
		return
	}
	if a.UserID != userID {
		a = nil
		err = domain.ErrUserUnauthorized
		return
	}
	return
}

// GetTransactions retrieves all transactions for a given account ID.
// Returns a slice of transactions or an error if the operation fails.
func (s *AccountService) GetTransactions(
	userID, accountID uuid.UUID,
) (txs []*domain.Transaction, err error) {
	uow, err := s.uowFactory()
	if err != nil {
		txs = nil
		return
	}

	txRepo, err := uow.TransactionRepository()
	if err != nil {
		txs = nil
		return
	}
	txs, err = txRepo.List(userID, accountID)
	if err != nil {
		txs = nil
		return
	}

	return
}

// GetBalance retrieves the current balance of the specified account.
// Returns the balance as a float64 or an error if the operation fails.
func (s *AccountService) GetBalance(
	userID, accountID uuid.UUID,
) (balance float64, err error) {
	uow, err := s.uowFactory()
	if err != nil {
		return
	}

	repo, err := uow.AccountRepository()
	if err != nil {
		return
	}
	a, err := repo.Get(accountID)
	if err != nil {
		return
	}

	balance, err = a.GetBalance(userID)
	return
}
