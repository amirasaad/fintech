package repository

type UnitOfWork interface {
	Begin() error
	Commit() error
	Rollback() error
	AccountRepository() AccountRepository
	TransactionRepository() TransactionRepository
	UserRepository() UserRepository
}
