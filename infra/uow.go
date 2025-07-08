package infra

import (
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/repository"
	"gorm.io/gorm"
)

type UoW struct {
	session *gorm.DB
	started bool
	cfg     config.DBConfig
}

func NewGormUoW(cfg config.DBConfig) (*UoW, error) {
	db, _ := NewDBConnection(cfg)
	return &UoW{
		session: db,
		started: false,
		cfg:     cfg,
	}, nil
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
	return err
}

func (u *UoW) AccountRepository() repository.AccountRepository {
	if !u.started {
		db, _ := NewDBConnection(u.cfg)
		return NewAccountRepository(db)
	}
	return NewAccountRepository(u.session)
}
func (u *UoW) TransactionRepository() repository.TransactionRepository {
	if !u.started {
		db, _ := NewDBConnection(u.cfg)
		return NewTransactionRepository(db)
	}
	return NewTransactionRepository(u.session)
}

func (u *UoW) UserRepository() repository.UserRepository {
	if !u.started {
		db, _ := NewDBConnection(u.cfg)
		return NewUserRepository(db)
	}
	return NewUserRepository(u.session)
}
