// Package service provides business logic for interacting with domain entities such as accounts and transactions.
// It defines the AccountService struct and its methods for creating accounts, depositing and withdrawing funds,
// retrieving account details, listing transactions, and checking account balances.
package service

import (
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"

	"github.com/google/uuid"
)

// AccountService provides methods to interact with accounts and transactions using a unit of work pattern.
type AccountService struct {
	uowFactory func() (repository.UnitOfWork, error)
}

// NewAccountService creates a new instance of AccountService with the provided unit of work factory.
func NewAccountService(uowFactory func() (repository.UnitOfWork, error)) *AccountService {
	return &AccountService{
		uowFactory: uowFactory,
	}
}

// CreateAccount creates a new account and persists it using the repository.
// Returns the created account or an error if the operation fails.
func (s *AccountService) CreateAccount() (*domain.Account, error) {
	uow, err := s.uowFactory()
	if err != nil {
		return nil, err
	}
	err = uow.Begin()
	if err != nil {
		return nil, err
	}

	a := domain.NewAccount()
	err = uow.AccountRepository().Create(a)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	err = uow.Commit()
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}
	return a, nil
}

// Deposit adds funds to the specified account and creates a transaction record.
// Returns the transaction or an error if the operation fails.
func (s *AccountService) Deposit(accountID uuid.UUID, amount float64) (*domain.Transaction, error) {
	uow, err := s.uowFactory()
	if err != nil {
		return nil, err
	}
	err = uow.Begin()
	if err != nil {
		slog.Error("Failed to begin transaction", slog.Any("error", err))
		return nil, err
	}

	a, err := uow.AccountRepository().Get(accountID)
	if err != nil {
		return nil, domain.ErrAccountNotFound
	}

	tx, err := a.Deposit(amount)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	err = uow.AccountRepository().Update(a)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	err = uow.TransactionRepository().Create(tx)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	err = uow.Commit()
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	return tx, nil
}

// Withdraw removes funds from the specified account and creates a transaction record.
// Returns the transaction or an error if the operation fails.
func (s *AccountService) Withdraw(accountID uuid.UUID, amount float64) (*domain.Transaction, error) {
	uow, err := s.uowFactory()
	if err != nil {
		return nil, err
	}
	err = uow.Begin()
	if err != nil {
		return nil, err
	}

	a, err := uow.AccountRepository().Get(accountID)
	if err != nil {
		_ = uow.Rollback()
		return nil, domain.ErrAccountNotFound
	}

	tx, err := a.Withdraw(amount)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	err = uow.AccountRepository().Update(a)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	err = uow.TransactionRepository().Create(tx)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	err = uow.Commit()
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	return tx, nil
}

// GetAccount retrieves an account by its ID.
// Returns the account or an error if not found.
func (s *AccountService) GetAccount(accountID uuid.UUID) (*domain.Account, error) {
	uow, err := s.uowFactory()
	if err != nil {
		return nil, err
	}
	a, err := uow.AccountRepository().Get(accountID)
	if err != nil {
		return nil, domain.ErrAccountNotFound
	}
	return a, nil
}

// GetTransactions retrieves all transactions for a given account ID.
// Returns a slice of transactions or an error if the operation fails.
func (s *AccountService) GetTransactions(accountID uuid.UUID) ([]*domain.Transaction, error) {
	uow, err := s.uowFactory()
	if err != nil {
		return nil, err
	}
	txs, err := uow.TransactionRepository().List(accountID)
	if err != nil {
		return nil, err
	}

	return txs, nil
}

// GetBalance retrieves the current balance of the specified account.
// Returns the balance as a float64 or an error if the operation fails.
func (s *AccountService) GetBalance(accountID uuid.UUID) (float64, error) {
	uow, err := s.uowFactory()
	if err != nil {
		return 0, err
	}
	a, err := uow.AccountRepository().Get(accountID)
	if err != nil {
		return 0, err
	}

	return a.GetBalance(), nil
}
