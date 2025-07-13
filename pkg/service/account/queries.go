package account

import (
	"context"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

// GetAccount retrieves an account by ID for the specified user.
func (s *Service) GetAccount(
	userID, accountID uuid.UUID,
) (
	account *account.Account,
	err error,
) {
	err = s.uow.Do(context.Background(), func(uow repository.UnitOfWork) error {
		repo, err := uow.AccountRepository()
		if err != nil {
			return err
		}
		account, err = repo.Get(accountID)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		account = nil
	}
	return
}

// GetTransactions retrieves all transactions for a specific account.
func (s *Service) GetTransactions(
	userID, accountID uuid.UUID,
) (
	transactions []*account.Transaction,
	err error,
) {
	err = s.uow.Do(context.Background(), func(uow repository.UnitOfWork) error {
		// First, validate that the account exists and belongs to the user
		accountRepo, err := uow.AccountRepository()
		if err != nil {
			return err
		}
		_, err = accountRepo.Get(accountID)
		if err != nil {
			return err
		}

		// Then, get the transactions
		transactionRepo, err := uow.TransactionRepository()
		if err != nil {
			return err
		}
		transactions, err = transactionRepo.List(userID, accountID)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		transactions = nil
	}
	return
}

// GetBalance retrieves the current balance of an account for the specified user.
func (s *Service) GetBalance(
	userID, accountID uuid.UUID,
) (
	balance float64,
	err error,
) {
	err = s.uow.Do(context.Background(), func(uow repository.UnitOfWork) error {
		repo, err := uow.AccountRepository()
		if err != nil {
			return err
		}
		account, err := repo.Get(accountID)
		if err != nil {
			return err
		}
		balance, err = account.GetBalance(userID)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		balance = 0
	}
	return
}
