package repository

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/repository"
	"gorm.io/gorm"
)

type UoW struct {
	session      *gorm.DB
	repoRegistry map[reflect.Type]func(*gorm.DB) interface{}
}

// NewGormUoW creates a new UoW with the given session (db or tx) and initializes the repository registry.
func NewGormUoW(db *gorm.DB) *UoW {
	return &UoW{
		session: db,
		repoRegistry: map[reflect.Type]func(*gorm.DB) interface{}{
			reflect.TypeOf((*repository.AccountRepository)(nil)).Elem(): func(db *gorm.DB) interface{} { return NewAccountRepository(db) },
			reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem(): func(db *gorm.DB) interface{} { return NewTransactionRepository(db) },
			reflect.TypeOf((*repository.UserRepository)(nil)).Elem(): func(db *gorm.DB) interface{} { return NewUserRepository(db) },
		},
	}
}

// GetRepository provides generic, type-safe access to repositories using a registry map.
// Example: repo, err := u.GetRepository[repository.UserRepository]()
func (u *UoW) GetRepository[T any]() (T, error) {
	var zero T
	t := reflect.TypeOf((*T)(nil)).Elem()
	constructor, ok := u.repoRegistry[t]
	if !ok {
		return zero, fmt.Errorf("unsupported repository type: %v", t)
	}
	repo := constructor(u.session)
	return repo.(T), nil
}
