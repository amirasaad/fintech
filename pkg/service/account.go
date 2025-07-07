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
func NewAccountService(
	uowFactory func() (repository.UnitOfWork, error),
) *AccountService {
	return &AccountService{
		uowFactory: uowFactory,
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
	err = uow.AccountRepository().Create(a)
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
	err = uow.AccountRepository().Create(account)
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
) (tx *domain.Transaction, err error) {
	uow, err := s.uowFactory()
	if err != nil {
		tx = nil
		return
	}
	err = uow.Begin()
	if err != nil {
		slog.Error("Failed to begin transaction", slog.Any("error", err))
		tx = nil
		return
	}

	a, err := uow.AccountRepository().Get(accountID)
	if err != nil {
		_ = uow.Rollback()
		tx = nil
		err = domain.ErrAccountNotFound
		return
	}
	tx, err = a.Deposit(userID, amount)
	if err != nil {
		_ = uow.Rollback()
		tx = nil
		return
	}

	err = uow.AccountRepository().Update(a)
	if err != nil {
		_ = uow.Rollback()
		tx = nil
		return
	}

	err = uow.TransactionRepository().Create(tx)
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
// Returns the transaction or an error if the operation fails.
func (s *AccountService) Withdraw(
	userID, accountID uuid.UUID,
	amount float64,
) (tx *domain.Transaction, err error) {
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

	a, err := uow.AccountRepository().Get(accountID)
	if err != nil {
		_ = uow.Rollback()
		tx = nil
		err = domain.ErrAccountNotFound
		return
	}

	tx, err = a.Withdraw(userID, amount)
	if err != nil {
		_ = uow.Rollback()
		tx = nil
		return
	}

	err = uow.AccountRepository().Update(a)
	if err != nil {
		_ = uow.Rollback()
		tx = nil
		return
	}

	err = uow.TransactionRepository().Create(tx)
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
	a, err = uow.AccountRepository().Get(accountID)
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
	txs, err = uow.TransactionRepository().List(userID, accountID)
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
	a, err := uow.AccountRepository().Get(accountID)
	if err != nil {
		return
	}

	balance, err = a.GetBalance(userID)
	return
}

// DepositWithCurrency adds funds to the account if the currency matches.
// Returns an error if the currency does not match.
func (s *AccountService) DepositWithCurrency(userID, accountID uuid.UUID, amount float64, currency string) (*domain.Transaction, error) {
	uow, err := s.uowFactory()
	if err != nil {
		return nil, err
	}
	err = uow.Begin()
	if err != nil {
		return nil, err
	}

	account, err := uow.AccountRepository().Get(accountID)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	// Delegate to Deposit
	tx, err := account.DepositWithCurrency(userID, amount, currency)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	err = uow.TransactionRepository().Create(tx)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	err = uow.AccountRepository().Update(account)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	err = uow.Commit()
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	tx.Currency = currency // Ensure transaction currency is set
	return tx, nil
}

// WithdrawWithCurrency withdraws funds from the account if the currency matches.
// Returns an error if the currency does not match.
func (s *AccountService) WithdrawWithCurrency(
	userID, accountID uuid.UUID,
	amount float64,
	currency string,
) (tx *domain.Transaction, err error) {
	uow, err := s.uowFactory()
	if err != nil {
		return nil, err
	}
	err = uow.Begin()
	if err != nil {
		return nil, err
	}

	account, err := uow.AccountRepository().Get(accountID)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	tx, err = account.WithdrawWithCurrency(userID, amount, currency)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	err = uow.AccountRepository().Update(account)
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
