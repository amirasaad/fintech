package infra

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ID       uuid.UUID `gorm:"type:uuid;primary_key"`
	Username string    `gorm:"uniqueIndex;not null;size:50;" validate:"required,min=3,max=50"`
	Email    string    `gorm:"uniqueIndex;not null;size:255;" validate:"required,email"`
	Password string    `gorm:"not null;" validate:"required,min=6,max=50"`
	Names    string    `json:"names"`
}

type Account struct {
	gorm.Model
	ID           uuid.UUID `gorm:"type:uuid;primary_key"`
	UserID       uuid.UUID `gorm:"type:uuid"`
	Balance      int64
	Currency     string `gorm:"type:varchar(3);not null;default:'USD'"`
	Transactions []Transaction
}

type Transaction struct {
	gorm.Model
	ID        uuid.UUID `gorm:"type:uuid;primary_key"`
	AccountID uuid.UUID `gorm:"type:uuid"`
	UserID    uuid.UUID `gorm:"type:uuid"`
	Amount    int64
	Currency  string `gorm:"type:varchar(3);not null;default:'USD'"`
	Balance   int64
}
