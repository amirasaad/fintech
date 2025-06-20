package infra

import (
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/internal/repository"
	"gorm.io/gorm"
)

type UoW struct {
	session *gorm.DB
	started bool
}

func NewGormUoW() (*UoW, error) {

	db, err := NewDBConnection()
	if err != nil {
		return nil, err
	}
	return &UoW{
		session: db,
		started: false,
	}, nil
}
func (u *UoW) Begin() error {
	slog.Info("Starting transaction")
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
	slog.Info("Committing transaction")
	if !u.started {
		slog.Info("Transaction not started, nothing to commit")
		return nil // No transaction to commit
	}
	err := u.session.Commit().Error
	if err != nil {
		slog.Error("Failed to commit transaction", slog.Any("error", err))
	} else {
		u.started = true
		slog.Info("Transaction committed successfully")
	}
	return err
}
func (u *UoW) Rollback() error {
	slog.Info("Rolling back transaction")
	if !u.started {
		slog.Info("Transaction not started, nothing to rollback")
		return nil // No transaction to rollback
	}
	err := u.session.Rollback().Error
	if err != nil {
		slog.Error("Failed to rollback transaction", slog.Any("error", err))
	} else {
		u.started = false
		slog.Info("Transaction rolled back successfully")
	}
	return err
}

func (u *UoW) AccountRepository() repository.AccountRepository {
	if !u.started {
		db, _ := NewDBConnection()
		return NewAccountRepository(db)
	}
	return NewAccountRepository(u.session)
}
func (u *UoW) TransactionRepository() repository.TransactionRepository {
	if !u.started {
		db, _ := NewDBConnection()
		return NewTransactionRepository(db)
	}
	return NewTransactionRepository(u.session)
}
