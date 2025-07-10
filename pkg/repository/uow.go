package repository

type UnitOfWork interface {
	Begin() error
	Commit() error
	Rollback() error
	AccountRepository() (AccountRepository, error)
	TransactionRepository() (TransactionRepository, error)
	UserRepository() (UserRepository, error)
}
