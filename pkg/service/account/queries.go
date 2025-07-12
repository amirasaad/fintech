package account

import (
	"context"
	"reflect"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

// GetAccount retrieves an account by its ID.
// Returns the account or an error if not found.
func (s *AccountService) GetAccount(
	userID, accountID uuid.UUID,
) (a *account.Account, err error) {
	s.logger.Info("GetAccount started", "userID", userID, "accountID", accountID)
	defer func() {
		if err != nil {
			s.logger.Error("GetAccount failed", "userID", userID, "accountID", accountID, "error", err)
		} else {
			s.logger.Info("GetAccount successful", "userID", userID, "accountID", accountID)
		}
	}()
	logger := s.logger.With("userID", userID, "accountID", accountID)
	logger.Info("GetAccount started")
	err = s.uow.Do(context.Background(), func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem())
		if err != nil {
			logger.Error("GetAccount failed: AccountRepository error", "error", err)
			return err
		}
		repo := repoAny.(repository.AccountRepository)
		aLocal, err := repo.Get(accountID)
		if err != nil {
			logger.Error("GetAccount failed: db error", "error", err)
			return err
		}
		if aLocal.UserID != userID {
			logger.Error("GetAccount failed: user unauthorized", "accountUserID", aLocal.UserID)
			return user.ErrUserUnauthorized
		}
		a = aLocal
		return nil
	})
	if err != nil {
		a = nil
	}
	return
}

// GetTransactions retrieves all transactions for a given account ID.
// Returns a slice of transactions or an error if the operation fails.
func (s *AccountService) GetTransactions(
	userID, accountID uuid.UUID,
) (txs []*account.Transaction, err error) {
	s.logger.Info("GetTransactions started", "userID", userID, "accountID", accountID)
	defer func() {
		if err != nil {
			s.logger.Error("GetTransactions failed", "userID", userID, "accountID", accountID, "error", err)
		} else {
			s.logger.Info("GetTransactions successful", "userID", userID, "accountID", accountID, "count", len(txs))
		}
	}()
	logger := s.logger.With("userID", userID, "accountID", accountID)
	logger.Info("GetTransactions started")
	err = s.uow.Do(context.Background(), func(uow repository.UnitOfWork) error {
		// First, verify the account exists and belongs to the user
		repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem())
		if err != nil {
			logger.Error("GetTransactions failed: AccountRepository error", "error", err)
			return err
		}
		repo := repoAny.(repository.AccountRepository)
		a, err := repo.Get(accountID)
		if err != nil {
			logger.Error("GetTransactions failed: account not found", "error", err)
			return err
		}
		if a == nil {
			logger.Error("GetTransactions failed: account not found")
			return account.ErrAccountNotFound
		}
		if a.UserID != userID {
			logger.Error("GetTransactions failed: user unauthorized", "accountUserID", a.UserID)
			return user.ErrUserUnauthorized
		}

		// Now get the transactions
		txRepoAny, err := uow.GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem())
		if err != nil {
			logger.Error("GetTransactions failed: TransactionRepository error", "error", err)
			return err
		}
		txRepo := txRepoAny.(repository.TransactionRepository)
		txs, err = txRepo.List(userID, accountID)
		if err != nil {
			logger.Error("GetTransactions failed: repo list error", "error", err)
			return err
		}
		return nil
	})
	if err != nil {
		txs = nil
	}
	return
}

// GetBalance retrieves the current balance of the specified account.
// Returns the balance as a float64 or an error if the operation fails.
func (s *AccountService) GetBalance(
	userID, accountID uuid.UUID,
) (balance float64, err error) {
	s.logger.Info("GetBalance started", "userID", userID, "accountID", accountID)
	defer func() {
		if err != nil {
			s.logger.Error("GetBalance failed", "userID", userID, "accountID", accountID, "error", err)
		} else {
			s.logger.Info("GetBalance successful", "userID", userID, "accountID", accountID, "balance", balance)
		}
	}()
	logger := s.logger.With("userID", userID, "accountID", accountID)
	logger.Info("GetBalance started")
	err = s.uow.Do(context.Background(), func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem())
		if err != nil {
			logger.Error("GetBalance failed: AccountRepository error", "error", err)
			return err
		}
		repo := repoAny.(repository.AccountRepository)
		a, err := repo.Get(accountID)
		if err != nil {
			logger.Error("GetBalance failed: AccountRepository.Get error", "error", err)
			return err
		}
		if a == nil {
			err = account.ErrAccountNotFound
			logger.Error("GetBalance failed:  ErrAccountNotFound")
			return err
		}
		balance, err = a.GetBalance(userID)
		if err != nil {
			logger.Error("GetBalance failed: domain error", "error", err)
			return err
		}
		return nil
	})
	if err != nil {
		balance = 0
	}
	return
}