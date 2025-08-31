package common

import (
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/amirasaad/fintech/pkg/repository/user"
)

var ErrInvalidRepositoryType = errors.New("invalid repository type")

func GetAccountRepository(
	uow repository.UnitOfWork,
	log *slog.Logger,
) (
	account.Repository,
	error,
) {
	accRepoAny, err := uow.GetRepository(
		(*account.Repository)(nil),
	)
	if err != nil {
		log.Error(
			"failed to get account repository",
			"error", err,
		)
		return nil, err
	}
	if accRepo, ok := accRepoAny.(account.Repository); ok {
		return accRepo, nil
	}
	return nil, ErrInvalidRepositoryType
}

func GetTransactionRepository(
	uow repository.UnitOfWork,
	log *slog.Logger,
) (
	transaction.Repository,
	error,
) {
	txRepoAny, err := uow.GetRepository(
		(*transaction.Repository)(nil),
	)
	if err != nil {
		log.Error(
			"failed to get transaction repository",
			"error", err,
		)
		return nil, err
	}
	if txRepo, ok := txRepoAny.(transaction.Repository); ok {

		return txRepo, nil
	}
	return nil, ErrInvalidRepositoryType
}

func GetUserRepository(
	uow repository.UnitOfWork,
	log *slog.Logger,
) (
	user.Repository,
	error,
) {
	userRepoAny, err := uow.GetRepository(
		(*user.Repository)(nil),
	)
	if err != nil {
		log.Error(
			"failed to get user repository",
			"error", err,
		)
		return nil, err
	}
	if userRepo, ok := userRepoAny.(user.Repository); ok {
		return userRepo, nil
	}
	return nil, ErrInvalidRepositoryType
}
