package repository

import (
	"context"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GenericRepository provides type-safe CRUD operations for any entity
type GenericRepository[T any] interface {
	// Basic CRUD operations
	Create(ctx context.Context, entity *T) error
	Get(ctx context.Context, id uuid.UUID) (*T, error)
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Query operations
	List(ctx context.Context) ([]*T, error)
	FindBy(ctx context.Context, query interface{}, args ...interface{}) ([]*T, error)
	FindOneBy(ctx context.Context, query interface{}, args ...interface{}) (*T, error)

	// Transaction support
	WithTransaction(tx *gorm.DB) GenericRepository[T]
}

// GenericRepositoryImpl implements GenericRepository for any entity type
type GenericRepositoryImpl[T any] struct {
	db *gorm.DB
}

// NewGenericRepository creates a new generic repository
func NewGenericRepository[T any](db *gorm.DB) GenericRepository[T] {
	return &GenericRepositoryImpl[T]{
		db: db,
	}
}

// Create saves a new entity to the database
func (r *GenericRepositoryImpl[T]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

// Get retrieves an entity by ID
func (r *GenericRepositoryImpl[T]) Get(ctx context.Context, id uuid.UUID) (*T, error) {
	var entity T
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&entity).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

// Update modifies an existing entity
func (r *GenericRepositoryImpl[T]) Update(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

// Delete removes an entity by ID
func (r *GenericRepositoryImpl[T]) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(new(T)).Error
}

// List retrieves all entities
func (r *GenericRepositoryImpl[T]) List(ctx context.Context) ([]*T, error) {
	var entities []*T
	err := r.db.WithContext(ctx).Find(&entities).Error
	return entities, err
}

// FindBy retrieves entities matching the query
func (r *GenericRepositoryImpl[T]) FindBy(ctx context.Context, query interface{}, args ...interface{}) ([]*T, error) {
	var entities []*T
	err := r.db.WithContext(ctx).Where(query, args...).Find(&entities).Error
	return entities, err
}

// FindOneBy retrieves a single entity matching the query
func (r *GenericRepositoryImpl[T]) FindOneBy(ctx context.Context, query interface{}, args ...interface{}) (*T, error) {
	var entity T
	err := r.db.WithContext(ctx).Where(query, args...).First(&entity).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

// WithTransaction returns a new repository instance using the provided transaction
func (r *GenericRepositoryImpl[T]) WithTransaction(tx *gorm.DB) GenericRepository[T] {
	return &GenericRepositoryImpl[T]{
		db: tx,
	}
}

// GenericUnitOfWorkWithGenerics provides type-safe repository access using generics
type GenericUnitOfWorkWithGenerics interface {
	// Transaction management
	Do(ctx context.Context, fn func(GenericUnitOfWorkWithGenerics) error) error
}

// GenericUnitOfWorkWithGenericsImpl implements the generic UOW interface
type GenericUnitOfWorkWithGenericsImpl struct {
	db *gorm.DB
	tx *gorm.DB
}

// NewGenericUnitOfWorkWithGenerics creates a new generic UOW instance
func NewGenericUnitOfWorkWithGenerics(db *gorm.DB) GenericUnitOfWorkWithGenerics {
	return &GenericUnitOfWorkWithGenericsImpl{
		db: db,
	}
}

// Do executes a function within a transaction
func (uow *GenericUnitOfWorkWithGenericsImpl) Do(ctx context.Context, fn func(GenericUnitOfWorkWithGenerics) error) error {
	return uow.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txnUow := &GenericUnitOfWorkWithGenericsImpl{
			db: uow.db,
			tx: tx,
		}
		return fn(txnUow)
	})
}

// Repository returns a generic repository for the specified entity type
func (uow *GenericUnitOfWorkWithGenericsImpl) Repository(entityType interface{}) interface{} {
	db := uow.db
	if uow.tx != nil {
		db = uow.tx
	}

	// This is a simplified version - in practice you'd want type-safe access
	switch entityType.(type) {
	case *domain.Account:
		return NewGenericRepository[domain.Account](db)
	case *domain.Transaction:
		return NewGenericRepository[domain.Transaction](db)
	case *domain.User:
		return NewGenericRepository[domain.User](db)
	default:
		return nil
	}
}
