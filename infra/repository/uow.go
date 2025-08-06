package repository

import (
	"context"
	"fmt"

	repoaccount "github.com/amirasaad/fintech/infra/repository/account"
	repotransaction "github.com/amirasaad/fintech/infra/repository/transaction"
	repouser "github.com/amirasaad/fintech/infra/repository/user"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/amirasaad/fintech/pkg/repository/user"
	"gorm.io/gorm"
)

// UoW provides transaction boundary and repository access in one abstraction.
//
// Why is GetRepository part of UoW?
// - Ensures all repositories use the same DB session/transaction for true atomicity.
// - Keeps service code clean and focused on business logic.
// - Centralizes repository wiring and registry for maintainability.
// - Prevents accidental use of the wrong DB session (which would break transactionality).
// - Is idiomatic for Go UoW patterns and easy to mock in tests.
type UoW struct {
	db      *gorm.DB
	tx      *gorm.DB
	repoMap map[any]func(*gorm.DB) any
}

// NewUoW creates a new UoW for the given *gorm.DB.
func NewUoW(db *gorm.DB) *UoW {
	return &UoW{
		db: db,
		repoMap: map[any]func(db *gorm.DB) any{
			(*account.Repository)(nil): func(db *gorm.DB) any {
				return repoaccount.New(db)
			},
			(*transaction.Repository)(nil): func(db *gorm.DB) any {
				return repotransaction.New(db)
			},
			(*user.Repository)(nil): func(db *gorm.DB) any {
				return repouser.New(db)
			},
		},
	}
}

// Do runs the given function in a transaction boundary, providing a UoW with repository access.
func (u *UoW) Do(ctx context.Context, fn func(uow repository.UnitOfWork) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txnUow := &UoW{
			db: u.db,
			tx: tx,
		}
		return fn(txnUow)
	})
}

// GetRepository provides generic, type-safe access to repositories using the transaction session.
// This method is maintained for backward compatibility
// but is deprecated in favor of type-safe methods.
//
// This method is part of UoW to guarantee that all repository operations within a transaction
// use the same DB session, ensuring atomicity and consistency. It also centralizes repository
// construction and makes testing and extension easier.
func (u *UoW) GetRepository(repoType any) (any, error) {
	// Use transaction DB if available, otherwise use main DB
	dbToUse := u.tx
	if dbToUse == nil {
		dbToUse = u.db
	}

	switch repoType {
	case (*account.Repository)(nil):
		return repoaccount.New(dbToUse), nil
	case (*transaction.Repository)(nil):
		return repotransaction.New(dbToUse), nil
	case (*user.Repository)(nil):
		return repouser.New(dbToUse), nil
	default:
		if repo, ok := u.repoMap[repoType]; ok {
			return repo(dbToUse), nil
		}
		return nil, fmt.Errorf(
			"unsupported repository type: %T, ", repoType)
	}
}
