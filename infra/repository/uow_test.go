package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestUoW_DoAndGetRepository(t *testing.T) {
	mockDb, mock, _ := sqlmock.New() // get the mock handle
	dialector := postgres.New(postgres.Config{
		Conn:       mockDb,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	assert.NoError(t, err)

	uow := NewUoW(db)

	// Expect transaction begin and commit
	mock.ExpectBegin()
	mock.ExpectCommit()

	err = uow.Do(context.Background(), func(txUow repository.UnitOfWork) error {
		repoAny, err := txUow.GetRepository((*repository.AccountRepository)(nil))
		assert.NoError(t, err)
		acctRepo := repoAny.(repository.AccountRepository)
		assert.NotNil(t, acctRepo)
		_, ok := acctRepo.(*accountRepository)
		assert.True(t, ok)

		repoAny, err = txUow.GetRepository((*repository.TransactionRepository)(nil))
		assert.NoError(t, err)
		txRepo := repoAny.(repository.TransactionRepository)
		assert.NotNil(t, txRepo)
		_, ok = txRepo.(*transactionRepository)
		assert.True(t, ok)

		repoAny, err = txUow.GetRepository((*repository.UserRepository)(nil))
		assert.NoError(t, err)
		userRepo := repoAny.(repository.UserRepository)
		assert.NotNil(t, userRepo)
		_, ok = userRepo.(*userRepository)
		assert.True(t, ok)

		return nil
	})
	assert.NoError(t, err)
}

func TestUoW_TypeSafeMethods(t *testing.T) {
	mockDb, mock, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{
		Conn:       mockDb,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	assert.NoError(t, err)

	uow := NewUoW(db)

	// Test type-safe methods outside transaction
	accountRepo, err := uow.AccountRepository()
	assert.NoError(t, err)
	assert.NotNil(t, accountRepo)

	transactionRepo, err := uow.TransactionRepository()
	assert.NoError(t, err)
	assert.NotNil(t, transactionRepo)

	userRepo, err := uow.UserRepository()
	assert.NoError(t, err)
	assert.NotNil(t, userRepo)

	// Test type-safe methods inside transaction
	mock.ExpectBegin()
	mock.ExpectCommit()

	err = uow.Do(context.Background(), func(txUow repository.UnitOfWork) error {
		accountRepo, err := txUow.AccountRepository()
		assert.NoError(t, err)
		assert.NotNil(t, accountRepo)

		transactionRepo, err := txUow.TransactionRepository()
		assert.NoError(t, err)
		assert.NotNil(t, transactionRepo)

		userRepo, err := txUow.UserRepository()
		assert.NoError(t, err)
		assert.NotNil(t, userRepo)

		return nil
	})
	assert.NoError(t, err)
}
