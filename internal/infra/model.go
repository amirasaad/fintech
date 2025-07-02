package infra

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ID       uuid.UUID `gorm:"primaryKey"`
	Username string    `gorm:"uniqueIndex;not null;size:50;" validate:"required,min=3,max=50" json:"username"`
	Email    string    `gorm:"uniqueIndex;not null;size:255;" validate:"required,email" json:"email"`
	Password string    `gorm:"not null;" validate:"required,min=6,max=50" json:"password"`
	Names    string    `json:"names"`
	Updated  time.Time `gorm:"autoUpdateTime"`
	Created  time.Time `gorm:"autoCreateTime"`
}

type Account struct {
	gorm.Model
	ID           uuid.UUID `gorm:"primaryKey"`
	UserID       uuid.UUID
	Balance      int64
	Updated      time.Time `gorm:"autoUpdateTime"`
	Created      time.Time `gorm:"autoCreateTime"`
	Transactions []Transaction
}

type Transaction struct {
	gorm.Model
	ID        uuid.UUID `gorm:"primaryKey"`
	AccountID uuid.UUID `json:"account_id"`
	Amount    int64
	Balance   int64
	Created   time.Time `gorm:"autoCreateTime"`
}
