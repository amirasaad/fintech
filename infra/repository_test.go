package infra

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestTransactionRepository_Create(t *testing.T) {
	assert := assert.New(t)
	mockDb, mock, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{
		Conn:       mockDb,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(err)

	transRepo := transactionRepository{db: db}
	userID := uuid.New()
	accountID := uuid.New()
	transaction := domain.NewTransactionFromData(uuid.New(), userID, accountID, 100, 100, time.Now())

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "transactions" (.+) VALUES (.+)`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = transRepo.Create(transaction)
	assert.NoError(err)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "transactions" (.+) VALUES (.+)`).
		WillReturnError(errors.New("create error"))
	mock.ExpectRollback()

	err = transRepo.Create(transaction)
	assert.Error(err)
}

func TestUserRepository_Create(t *testing.T) {
	assert := assert.New(t)
	mockDb, mock, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{
		Conn:       mockDb,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(err)

	userRepo := userRepository{db: db}
	user, _ := domain.NewUser("testuser", "test@example.com", "password")

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "users" (.+) VALUES (.+)`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = userRepo.Create(user)
	assert.NoError(err)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "users" (.+) VALUES (.+)`).
		WillReturnError(errors.New("create error"))
	mock.ExpectRollback()

	err = userRepo.Create(user)
	assert.Error(err)
}

func TestAccountRepository_Get(t *testing.T) {
	assert := assert.New(t)
	mockDb, mock, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{
		Conn:       mockDb,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(err)

	accRepo := accountRepository{db: db}
	userID := uuid.New()
	accountID := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "user_id", "created_at", "updated_at", "balance"}).
		AddRow(accountID, userID, "2023-01-01 00:00:00", "2023-01-01 00:00:00", 100)
	mock.ExpectQuery(`SELECT \* FROM "accounts" WHERE "accounts"\."id" = \$1 ORDER BY "accounts"\."id" LIMIT \$2`).
		WithArgs(accountID, 1).WillReturnRows(rows)

	account, err := accRepo.Get(accountID)
	assert.NoError(err)
	assert.NotNil(account)
	assert.Equal(accountID, account.ID)

	mock.ExpectQuery(`SELECT \* FROM "accounts" WHERE "accounts"\."id" = \$1 ORDER BY "accounts"\."id" LIMIT \$2`).
		WithArgs(sqlmock.AnyArg(), 1).WillReturnError(gorm.ErrRecordNotFound)
	account, err = accRepo.Get(uuid.New())
	assert.Error(err)
	assert.Nil(account)
}

func TestAccountRepository_Update(t *testing.T) {
	assert := assert.New(t)
	mockDb, mock, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{
		Conn:       mockDb,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(err)

	accRepo := accountRepository{db: db}
	userID := uuid.New()
	account := domain.NewAccount(userID)
	account.Balance = 200

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "accounts" SET "user_id"=$1,"balance"=$2,"created_at"=$3,"updated_at"=$4 WHERE "id" = $5`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = accRepo.Update(account)
	assert.NoError(err)

	mock.ExpectBegin()
	mock.ExpectExec(
		`UPDATE "accounts" SET "user_id"=$1,"balance"=$2,"created_at"=$3,"updated_at"=$4 WHERE "id" = $5`).
		WillReturnError(errors.New("update error"))
	mock.ExpectRollback()

	err = accRepo.Update(account)
	assert.Error(err)
}

func TestAccountRepository_Delete(t *testing.T) {
	assert := assert.New(t)
	mockDb, mock, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{
		Conn:       mockDb,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(err)

	accRepo := accountRepository{db: db}
	accountID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "accounts" WHERE "id" = \$1`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = accRepo.Delete(accountID)
	assert.NoError(err)

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "accounts" WHERE "id" = \$1`).
		WillReturnError(errors.New("delete error"))
	mock.ExpectRollback()

	err = accRepo.Delete(uuid.New())
	assert.Error(err)
}
