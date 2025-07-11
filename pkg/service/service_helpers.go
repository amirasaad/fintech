package service

import (
	"log/slog"

	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/decorator"
)

// withRepoTransaction is a generic DRY helper for transaction, repository access, and logging.
//
// Parameters:
//   - logger: the service's logger
//   - transaction: the transaction decorator
//   - uowFactory: function to create a UnitOfWork
//   - opName: operation name for logging
//   - logFields: map of log fields for context
//   - repoGetter: function to get the desired repository from the UnitOfWork
//   - fn: business logic callback, receives the repository
//
// Returns:
//   - error: any error encountered during the operation
//
// Usage Example (UserRepository):
//
//   err := withRepoTransaction(
//       s.logger,
//       s.transaction,
//       s.uowFactory,
//       "CreateUser",
//       map[string]any{"username": username},
//       func(uow repository.UnitOfWork) (repository.UserRepository, error) {
//           return uow.UserRepository()
//       },
//       func(repo repository.UserRepository) error {
//           return repo.Create(user)
//       },
//   )
//
// Usage Example (AccountRepository):
//
//   err := withRepoTransaction(
//       s.logger,
//       s.transaction,
//       s.uowFactory,
//       "CreateAccount",
//       map[string]any{"userID": userID},
//       func(uow repository.UnitOfWork) (repository.AccountRepository, error) {
//           return uow.AccountRepository()
//       },
//       func(repo repository.AccountRepository) error {
//           return repo.Create(account)
//       },
//   )
//
func withRepoTransaction[Repo any](
	logger *slog.Logger,
	transaction decorator.TransactionDecorator,
	uowFactory func() (repository.UnitOfWork, error),
	opName string,
	logFields map[string]any,
	repoGetter func(uow repository.UnitOfWork) (Repo, error),
	fn func(repo Repo) error,
) (err error) {
	logger.Info(opName+" started", logFields)
	defer func() {
		if err != nil {
			logger.Error(opName+" failed", logFields, "error", err)
		} else {
			logger.Info(opName+" successful", logFields)
		}
	}()
	err = transaction.Execute(func() error {
		uow, err := uowFactory()
		if err != nil {
			return err
		}
		repo, err := repoGetter(uow)
		if err != nil {
			return err
		}
		return fn(repo)
	})
	return
}