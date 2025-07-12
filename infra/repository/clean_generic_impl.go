package repository

import (
	"context"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CleanGenericRepositoryImpl implements CleanGenericRepository for GORM
type CleanGenericRepositoryImpl[T any] struct {
	db *gorm.DB
}

// NewCleanGenericRepository creates a new clean generic repository
func NewCleanGenericRepository[T any](db *gorm.DB) repository.CleanGenericRepository[T] {
	return &CleanGenericRepositoryImpl[T]{
		db: db,
	}
}

// Create saves a new entity to the database
func (r *CleanGenericRepositoryImpl[T]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

// Get retrieves an entity by ID
func (r *CleanGenericRepositoryImpl[T]) Get(ctx context.Context, id uuid.UUID) (*T, error) {
	var entity T
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&entity).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

// Update modifies an existing entity
func (r *CleanGenericRepositoryImpl[T]) Update(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

// Delete removes an entity by ID
func (r *CleanGenericRepositoryImpl[T]) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(new(T)).Error
}

// List retrieves all entities
func (r *CleanGenericRepositoryImpl[T]) List(ctx context.Context) ([]*T, error) {
	var entities []*T
	err := r.db.WithContext(ctx).Find(&entities).Error
	return entities, err
}

// FindBy retrieves entities matching the query
func (r *CleanGenericRepositoryImpl[T]) FindBy(ctx context.Context, query interface{}, args ...interface{}) ([]*T, error) {
	var entities []*T
	err := r.db.WithContext(ctx).Where(query, args...).Find(&entities).Error
	return entities, err
}

// FindOneBy retrieves a single entity matching the query
func (r *CleanGenericRepositoryImpl[T]) FindOneBy(ctx context.Context, query interface{}, args ...interface{}) (*T, error) {
	var entity T
	err := r.db.WithContext(ctx).Where(query, args...).First(&entity).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

// GORMTransactionManager implements TransactionManager for GORM
type GORMTransactionManager struct {
	db *gorm.DB
}

// NewGORMTransactionManager creates a new GORM transaction manager
func NewGORMTransactionManager(db *gorm.DB) repository.TransactionManager {
	return &GORMTransactionManager{
		db: db,
	}
}

// ExecuteInTransaction runs a function within a GORM transaction
func (tm *GORMTransactionManager) ExecuteInTransaction(ctx context.Context, fn func() error) error {
	return tm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn()
	})
}

// NewCleanUnitOfWorkWithGORM creates a new clean UOW with GORM implementation
func NewCleanUnitOfWorkWithGORM(db *gorm.DB) repository.CleanUnitOfWork {
	return repository.NewCleanUnitOfWork(
		NewCleanGenericRepository[account.Account](db),
		NewCleanGenericRepository[account.Transaction](db),
		NewCleanGenericRepository[user.User](db),
		NewGORMTransactionManager(db),
	)
}

// Example usage:
//
// func main() {
//     db := // ... initialize GORM DB
//     uow := NewCleanUnitOfWorkWithGORM(db)
//
//     accountService := &AccountService{
//         uow: uow,
//         converter: converter,
//         logger: logger,
//     }
//
//     // Use the service - no infrastructure coupling in business logic!
//     err := accountService.Deposit(userID, accountID, amount, currency)
// }
