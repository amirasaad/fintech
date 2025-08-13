package account

import (
	"github.com/amirasaad/fintech/infra/repository/transaction"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Account represents an account record in the database.
type Account struct {
	gorm.Model
	ID           uuid.UUID `gorm:"type:uuid;primary_key"`
	UserID       uuid.UUID `gorm:"type:uuid"`
	Balance      int64
	Currency     string `gorm:"type:varchar(3);not null;default:'USD'"`
	Transactions []transaction.Transaction
}

// TableName specifies the table name for the Account model.
func (Account) TableName() string {
	return "accounts"
}
