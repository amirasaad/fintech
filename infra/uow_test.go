package infra

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestUnitOfWork(t *testing.T) {
	mockDb, _, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{
		Conn:       mockDb,
		DriverName: "postgres",
	})
	_, _ = gorm.Open(dialector, &gorm.Config{})
	uow, err := NewGormUoW(config.DBConfig{Url: "postgres:"})

	assert.NoError(t, err)

	// Test Accounts
	accounts := uow.AccountRepository()
	assert.IsType(t, &accountRepository{}, accounts)

	// Test Transactions
	transactions := uow.TransactionRepository()
	assert.IsType(t, &transactionRepository{}, transactions)
}

func TestUoW_Begin(t *testing.T) {
	assert := assert.New(t)
	mockDb, mock, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{
		Conn:       mockDb,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	assert.NoError(err)

	// Create UoW with the mock database directly
	uow := &UoW{
		session: db,
		started: false,
		cfg:     config.DBConfig{Url: "postgres://mock"},
	}

	mock.ExpectBegin()
	err = uow.Begin()
	assert.NoError(err)
	assert.True(uow.started)

	// Test already started
	err = uow.Begin()
	assert.NoError(err)

	// Test begin error
	mock.ExpectBegin().WillReturnError(errors.New("begin error"))
	uow.started = false // Reset started flag
	err = uow.Begin()
	assert.Error(err)
}

func TestUoW_Commit(t *testing.T) {
	assert := assert.New(t)
	mockDb, mock, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{
		Conn:       mockDb,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	assert.NoError(err)

	uow := &UoW{
		session: db,
		started: true,
		cfg:     config.DBConfig{Url: "postgres://mock"},
	}
	mock.ExpectBegin()
	db.Begin()
	mock.ExpectCommit().WillReturnError(errors.New("commit error"))
	err = uow.Commit()
	assert.Error(err)
	assert.False(uow.started) // Should be false after commit error
}

func TestUoW_Rollback(t *testing.T) {
	assert := assert.New(t)
	mockDb, mock, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{
		Conn:       mockDb,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	assert.NoError(err)

	// Test not started
	uow1 := &UoW{
		session: db,
		started: false,
		cfg:     config.DBConfig{Url: "postgres://mock"},
	}
	err = uow1.Rollback()
	assert.NoError(err)

	// Test rollback success
	uow2 := &UoW{
		session: db,
		started: false,
		cfg:     config.DBConfig{Url: "postgres://mock"},
	}
	mock.ExpectBegin()
	_ = uow2.Begin()
	mock.ExpectRollback()
	err = uow2.Rollback()
	assert.NoError(err)
	assert.False(uow2.started)

	// Test rollback error
	uow3 := &UoW{
		session: db,
		started: false,
		cfg:     config.DBConfig{Url: "postgres://mock"},
	}
	mock.ExpectBegin()
	_ = uow3.Begin()
	mock.ExpectRollback().WillReturnError(errors.New("rollback error"))
	err = uow3.Rollback()
	assert.Error(err)
	assert.False(uow3.started) // Should be false after rollback error
}

func TestUoW_Simple(t *testing.T) {
	assert := assert.New(t)
	mockDb, mock, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{
		Conn:       mockDb,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	assert.NoError(err)

	// Create UoW with the mock database directly
	uow := &UoW{
		session: db,
		started: false,
		cfg:     config.DBConfig{Url: "postgres://mock"},
	}

	// Test basic functionality without accessing unexported fields
	mock.ExpectBegin()
	err = uow.Begin()
	assert.NoError(err)

	mock.ExpectCommit()
	err = uow.Commit()
	assert.NoError(err)
}
