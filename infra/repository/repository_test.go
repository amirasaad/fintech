package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestTransactionRepository_Create(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
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
	transaction := account.NewTransactionFromData(uuid.New(), userID, accountID, 100, 100, "USD", time.Now(), nil, nil, nil)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "transactions" (.+) VALUES (.+) RETURNING "id"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(transaction.ID))
	mock.ExpectCommit()

	err = transRepo.Create(transaction)
	assert.NoError(err)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "transactions" (.+) VALUES (.+) RETURNING "id"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("create error"))
	mock.ExpectRollback()

	err = transRepo.Create(transaction)
	require.Error(err)
}

func TestUserRepository_Create(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
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
	user, _ := user.NewUser("testuser", "test@example.com", "password")

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
	require.Error(err)
}

func TestAccountRepository_Get(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
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
		AddRow(accountID, userID, time.Now().UTC(), time.Now().UTC(), 100)
	mock.ExpectQuery(`SELECT \* FROM "accounts" WHERE "accounts"\."id" = \$1 ORDER BY "accounts"\."id" LIMIT \$2`).
		WithArgs(accountID, 1).WillReturnRows(rows)

	account, err := accRepo.Get(accountID)
	assert.NoError(err)
	assert.NotNil(account)
	assert.Equal(accountID, account.ID)

	mock.ExpectQuery(`SELECT \* FROM "accounts" WHERE "accounts"\."id" = \$1 ORDER BY "accounts"\."id" LIMIT \$2`).
		WithArgs(sqlmock.AnyArg(), 1).WillReturnError(gorm.ErrRecordNotFound)
	account, err = accRepo.Get(uuid.New())
	require.Error(err)
	assert.Nil(account)
}
