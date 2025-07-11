package repository

import (
	"context"
	"reflect"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestUoW_DoAndGetRepository(t *testing.T) {
	mockDb, _, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{
		Conn:       mockDb,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	assert.NoError(t, err)

	uow := NewUoW(db)

	err = uow.Do(context.Background(), func(txUow repository.UnitOfWork) error {
		repoAny, err := txUow.GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem())
		assert.NoError(t, err)
		acctRepo := repoAny.(repository.AccountRepository)
		assert.NotNil(t, acctRepo)
		_, ok := acctRepo.(*accountRepository)
		assert.True(t, ok)

		repoAny, err = txUow.GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem())
		assert.NoError(t, err)
		txRepo := repoAny.(repository.TransactionRepository)
		assert.NotNil(t, txRepo)
		_, ok = txRepo.(*transactionRepository)
		assert.True(t, ok)

		repoAny, err = txUow.GetRepository(reflect.TypeOf((*repository.UserRepository)(nil)).Elem())
		assert.NoError(t, err)
		userRepo := repoAny.(repository.UserRepository)
		assert.NotNil(t, userRepo)
		_, ok = userRepo.(*userRepository)
		assert.True(t, ok)

		return nil
	})
	assert.NoError(t, err)
}
