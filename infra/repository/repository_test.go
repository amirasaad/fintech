package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestTransactionRepository_Create(t *testing.T) {
	require := require.New(t)
	mockDb, mock, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{
		Conn:       mockDb,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(err)

	transRepo := transactionRepository{db: db}
	userID := uuid.New()
	accountID := uuid.New()
	amount, _ := money.New(100, currency.USD)
	balance, _ := money.New(100, currency.USD)
	transaction := account.NewTransactionFromData(uuid.New(), userID, accountID, amount, balance, account.MoneySourceInternal, time.Now())

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "transactions" (.+) VALUES (.+) RETURNING "id"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(transaction.ID))
	mock.ExpectCommit()

	err = transRepo.Create(transaction, &common.ConversionInfo{})
	require.NoError(err)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "transactions" (.+) VALUES (.+) RETURNING "id"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("create error"))
	mock.ExpectRollback()

	err = transRepo.Create(transaction, &common.ConversionInfo{})
	require.Error(err)
}

func TestUserRepository_Create(t *testing.T) {
	require := require.New(t)
	mockDb, mock, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{
		Conn:       mockDb,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(err)

	userRepo := userRepository{db: db}
	user, _ := user.NewUser("testuser", "test@example.com", "password")

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "users" (.+) VALUES (.+)`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = userRepo.Create(user)
	require.NoError(err)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "users" (.+) VALUES (.+)`).
		WillReturnError(errors.New("create error"))
	mock.ExpectRollback()

	err = userRepo.Create(user)
	require.Error(err)
}

func TestAccountRepository_Get(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	mockDb, mock, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{
		Conn:       mockDb,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(err)

	accRepo := accountRepository{db: db}
	userID := uuid.New()
	accountID := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "user_id", "created_at", "updated_at", "balance"}).
		AddRow(accountID, userID, time.Now().UTC(), time.Now().UTC(), 100)
	mock.ExpectQuery(`SELECT \* FROM "accounts" WHERE "accounts"\."id" = \$1 AND "accounts"\."deleted_at" IS NULL ORDER BY "accounts"\."id" LIMIT \$2`).
		WithArgs(accountID, 1).WillReturnRows(rows)

	account, err := accRepo.Get(accountID)
	require.NoError(err)
	assert.NotNil(account)
	require.Equal(accountID, account.ID)

	mock.ExpectQuery(`SELECT \* FROM "accounts" WHERE "accounts"\."id" = \$1 AND "accounts"\."deleted_at" IS NULL ORDER BY "accounts"\."id" LIMIT \$2`).
		WithArgs(sqlmock.AnyArg(), 1).WillReturnError(gorm.ErrRecordNotFound)
	account, err = accRepo.Get(uuid.New())
	require.Error(err)
	assert.Nil(account)
}
