package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Account struct {
	gorm.Model
	ID           uuid.UUID `gorm:"primaryKey"`
	Balance      int64
	Updated      int64 `gorm:"autoUpdateTime"`
	Created      int64 `gorm:"autoCreateTime"`
	Transactions []Transaction
}

type Transaction struct {
	gorm.Model
	ID        uuid.UUID `gorm:"primaryKey"`
	AccountID uuid.UUID `json:"account_id"`
	Amount    int64
	Created   int64 `gorm:"autoCreateTime"`
}
