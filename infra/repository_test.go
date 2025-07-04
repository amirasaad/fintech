package infra

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestRepository_AccountCreate(t *testing.T) {
	mockDb, mock, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{
		Conn:       mockDb,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(t, err)

	accRepo := accountRepository{db: db}
	userID := uuid.New()
	acc := domain.NewAccount(userID)
	_ = accRepo.Create(domain.NewAccount(userID))

	_, _ = accRepo.Get(acc.ID)
	rows := sqlmock.NewRows([]string{"id", "user_id", "created_at", "updated_at", "balance"}).AddRow(acc.ID, userID, "2023-01-01 00:00:00", "2023-01-01 00:00:00", 0)
	mock.ExpectQuery(`SELECT`).WillReturnRows(rows)
}
