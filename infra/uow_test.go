package infra

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
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
	db, _ := gorm.Open(dialector, &gorm.Config{})
	uow, err := NewGormUoW(db)

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

	uow, err := NewGormUoW(db)
	assert.NoError(err)

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

	uow, err := NewGormUoW(db)
	assert.NoError(err)

	// Test not started
	err = uow.Commit()
	assert.NoError(err)

	// Test commit success
	mock.ExpectBegin()
	_ = uow.Begin()
	mock.ExpectCommit()
	err = uow.Commit()
	assert.NoError(err)
	assert.True(uow.started)

	// Test commit error
	mock.ExpectBegin()
	_ = uow.Begin()
	mock.ExpectCommit().WillReturnError(errors.New("commit error"))
	err = uow.Commit()
	assert.Error(err)
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

	uow, err := NewGormUoW(db)
	assert.NoError(err)

	// Test not started
	err = uow.Rollback()
	assert.NoError(err)

	// Test rollback success
	mock.ExpectBegin()
	_ = uow.Begin()
	mock.ExpectRollback()
	err = uow.Rollback()
	assert.NoError(err)
	assert.False(uow.started)

	// Test rollback error
	mock.ExpectBegin()
	_ = uow.Begin()
	mock.ExpectRollback().WillReturnError(errors.New("rollback error"))
}
