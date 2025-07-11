package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
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

	err = uow.Do(context.Background(), func(txUow UnitOfWork) error {
		acctRepo, err := txUow.GetRepository[AccountRepository]()
		assert.NoError(t, err)
		assert.NotNil(t, acctRepo)
		_, ok := acctRepo.(*accountRepository)
		assert.True(t, ok)

		txRepo, err := txUow.GetRepository[TransactionRepository]()
		assert.NoError(t, err)
		assert.NotNil(t, txRepo)
		_, ok = txRepo.(*transactionRepository)
		assert.True(t, ok)

		userRepo, err := txUow.GetRepository[UserRepository]()
		assert.NoError(t, err)
		assert.NotNil(t, userRepo)
		_, ok = userRepo.(*userRepository)
		assert.True(t, ok)

		return nil
	})
	assert.NoError(t, err)
}
