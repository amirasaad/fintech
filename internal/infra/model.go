package infra

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Account struct {
	gorm.Model
	ID           uuid.UUID `gorm:"primaryKey"`
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
