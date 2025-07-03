package infra

import (
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
