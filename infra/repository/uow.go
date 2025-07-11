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
	baseDB      *gorm.DB // shared, non-transactional connection
	session     *gorm.DB // transactional session
	started     bool
	cfg         config.DBConfig
	appEnv      string
	repoRegistry map[reflect.Type]func(*gorm.DB) interface{}
}

// NewGormUoW now accepts a *gorm.DB instance and does not create a new connection
// It also initializes the repository registry for generic GetRepository.
func NewGormUoW(db *gorm.DB) *UoW {
	return &UoW{
		baseDB:  db,
		session: db,
		started: false,
		repoRegistry: map[reflect.Type]func(*gorm.DB) interface{}{
			reflect.TypeOf((*repository.AccountRepository)(nil)).Elem(): func(db *gorm.DB) interface{} { return NewAccountRepository(db) },
			reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem(): func(db *gorm.DB) interface{} { return NewTransactionRepository(db) },
			reflect.TypeOf((*repository.UserRepository)(nil)).Elem(): func(db *gorm.DB) interface{} { return NewUserRepository(db) },
		},
	}
}

func (u *UoW) Begin() error {
	slog.Info("UoW Begin()")
	if u.session == nil {
		slog.Error("Session is nil, cannot start transaction")
		return fmt.Errorf("session is nil")
	}
	if u.started {
		slog.Info("Transaction already started")
		return nil // Transaction already started
	}
	slog.Info("Starting new transaction")
	tx := u.session.Begin()
	if tx.Error != nil {
		slog.Error("Failed to start transaction", slog.Any("error", tx.Error))
		return tx.Error
	}
	slog.Info("Transaction started successfully")
	u.session = tx
	u.started = true
	return nil
}

func (u *UoW) Commit() error {
	slog.Info("UoW Commit()")
	if !u.started {
		slog.Info("Transaction not started, nothing to commit")
		return nil // No transaction to commit
	}
	err := u.session.Commit().Error
	if err != nil {
		slog.Error("Failed to commit transaction", slog.Any("error", err))
	} else {
		slog.Info("Transaction committed successfully")
	}
	u.started = false
	// After commit, reset session to baseDB
	u.session = u.baseDB
	return err
}

func (u *UoW) Rollback() error {
	slog.Info("UoW Rollback()")
	if !u.started {
		slog.Info("Transaction not started, nothing to rollback")
		return nil // No transaction to rollback
	}
	err := u.session.Rollback().Error
	if err != nil {
		slog.Error("Failed to rollback transaction", slog.Any("error", err))
	} else {
		slog.Info("Transaction rolled back successfully")
	}
	u.started = false // Always reset the flag, regardless of success/failure
	// After rollback, reset session to baseDB
	u.session = u.baseDB
	return err
}

func (u *UoW) AccountRepository() (repository.AccountRepository, error) {
	if u.started {
		if u.session == nil {
			return nil, fmt.Errorf("transactional session is nil")
		}
		return NewAccountRepository(u.session), nil
	}
	if u.baseDB == nil {
		return nil, fmt.Errorf("baseDB is nil")
	}
	return NewAccountRepository(u.baseDB), nil
}

func (u *UoW) TransactionRepository() (repository.TransactionRepository, error) {
	if u.started {
		if u.session == nil {
			return nil, fmt.Errorf("transactional session is nil")
		}
		return NewTransactionRepository(u.session), nil
	}
	if u.baseDB == nil {
		return nil, fmt.Errorf("baseDB is nil")
	}
	return NewTransactionRepository(u.baseDB), nil
}

func (u *UoW) UserRepository() (repository.UserRepository, error) {
	if u.started {
		if u.session == nil {
			return nil, fmt.Errorf("transactional session is nil")
		}
		return NewUserRepository(u.session), nil
	}
	if u.baseDB == nil {
		return nil, fmt.Errorf("baseDB is nil")
	}
	return NewUserRepository(u.baseDB), nil
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
	var db *gorm.DB
	if u.started {
		db = u.session
	} else {
		db = u.baseDB
	}
	repo := constructor(db)
	return repo.(T), nil
}
