package account

import (
	"context"

	"github.com/amirasaad/fintech/pkg/dto"
	repoaccount "github.com/amirasaad/fintech/pkg/repository/account"
	transactionrepo "github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
)

// GetAccount retrieves an account by ID for the specified user.
func (s *Service) GetAccount(
	ctx context.Context,
	userID, accountID uuid.UUID,
) (
	account *dto.AccountRead,
	err error,
) {
	repoAny, err := s.uow.GetRepository((*repoaccount.Repository)(nil))
	if err != nil {
		return
	}
	repo, ok := repoAny.(repoaccount.Repository)
	if !ok {
		return
	}
	account, err = repo.Get(ctx, accountID)
	return
}

// GetTransactions retrieves all transactions for a specific account.
func (s *Service) GetTransactions(
	ctx context.Context,
	userID, accountID uuid.UUID,
) (
	transactions []*dto.TransactionRead,
	err error,
) {
	// First, validate that the account exists and belongs to the user
	accountRepoAny, err := s.uow.GetRepository((*repoaccount.Repository)(nil))
	if err != nil {
		return
	}
	accountRepo, ok := accountRepoAny.(repoaccount.Repository)
	if !ok {
		return
	}
	_, err = accountRepo.Get(ctx, accountID)
	if err != nil {
		return
	}

	// Then, get the transactions
	transactionRepoAny, err := s.uow.GetRepository((*transactionrepo.Repository)(nil))
	if err != nil {
		return
	}
	transactionRepo, ok := transactionRepoAny.(transactionrepo.Repository)
	if !ok {
		return
	}
	transactions, err = transactionRepo.ListByAccount(ctx, accountID)
	return
}

// GetBalance retrieves the current balance of an account for the specified user.
func (s *Service) GetBalance(
	ctx context.Context,
	userID, accountID uuid.UUID,
) (
	balance float64,
	err error,
) {
	repoAny, err := s.uow.GetRepository((*repoaccount.Repository)(nil))
	if err != nil {
		return
	}
	repo, ok := repoAny.(repoaccount.Repository)
	if !ok {
		return
	}
	acc, err := repo.Get(ctx, accountID)
	if err != nil {
		return
	}

	if acc.UserID != userID {
		return
	}

	balance = acc.Balance
	return
}
