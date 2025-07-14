package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestUoW_DoAndGetRepository(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	mockDb, mock, _ := sqlmock.New() // get the mock handle
	dialector := postgres.New(postgres.Config{
		Conn:       mockDb,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(err)

	uow := NewUoW(db)

	// Expect transaction begin and commit
	mock.ExpectBegin()
	mock.ExpectCommit()

	err = uow.Do(context.Background(), func(txUow repository.UnitOfWork) error {
		repoAny, err := txUow.GetRepository((*repository.AccountRepository)(nil))
		require.NoError(err)
		acctRepo := repoAny.(repository.AccountRepository)
		assert.NotNil(acctRepo)
		_, ok := acctRepo.(*accountRepository)
		assert.True(ok)

		repoAny, err = txUow.GetRepository((*repository.TransactionRepository)(nil))
		require.NoError(err)
		txRepo := repoAny.(repository.TransactionRepository)
		assert.NotNil(txRepo)
		_, ok = txRepo.(*transactionRepository)
		assert.True(ok)

		repoAny, err = txUow.GetRepository((*repository.UserRepository)(nil))
		require.NoError(err)
		userRepo := repoAny.(repository.UserRepository)
		assert.NotNil(userRepo)
		_, ok = userRepo.(*userRepository)
		assert.True(ok)

		return nil
	})
	assert.NoError(err)
}

func TestUoW_TypeSafeMethods(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	mockDb, mock, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{
		Conn:       mockDb,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(err)

	uow := NewUoW(db)

	// Test type-safe methods outside transaction
	accountRepo, err := uow.AccountRepository()
	require.NoError(err)
	assert.NotNil(accountRepo)

	transactionRepo, err := uow.TransactionRepository()
	require.NoError(err)
	assert.NotNil(transactionRepo)

	userRepo, err := uow.UserRepository()
	require.NoError(err)
	assert.NotNil(userRepo)

	// Test type-safe methods inside transaction
	mock.ExpectBegin()
	mock.ExpectCommit()

	err = uow.Do(context.Background(), func(txUow repository.UnitOfWork) error {
		accountRepo, err := txUow.AccountRepository()
		require.NoError(err)
		assert.NotNil(accountRepo)

		transactionRepo, err := txUow.TransactionRepository()
		require.NoError(err)
		assert.NotNil(transactionRepo)

		userRepo, err := txUow.UserRepository()
		require.NoError(err)
		assert.NotNil(userRepo)

		return nil
	})
	assert.NoError(err)
}
