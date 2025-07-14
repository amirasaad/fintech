package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a user record in the database.
type User struct {
	gorm.Model
	ID       uuid.UUID `gorm:"type:uuid;primary_key"`
	Username string    `gorm:"uniqueIndex;not null;size:50;" validate:"required,min=3,max=50"`
	Email    string    `gorm:"uniqueIndex;not null;size:255;" validate:"required,email"`
	Password string    `gorm:"not null;" validate:"required,min=6,max=50"`
	Names    string    `json:"names"`
}

// Account represents an account record in the database.
type Account struct {
	gorm.Model
	ID           uuid.UUID `gorm:"type:uuid;primary_key"`
	UserID       uuid.UUID `gorm:"type:uuid"`
	Balance      int64
	Currency     string `gorm:"type:varchar(3);not null;default:'USD'"`
	Transactions []Transaction
}

// Transaction represents a persisted financial transaction.
type Transaction struct {
	gorm.Model
	ID        uuid.UUID `gorm:"type:uuid;primary_key"`
	AccountID uuid.UUID `gorm:"type:uuid"`
	UserID    uuid.UUID `gorm:"type:uuid"`
	Amount    int64
	Currency  string `gorm:"type:varchar(3);not null;default:'USD'"`
	Balance   int64

	// Conversion fields (nullable when no conversion occurs)
	OriginalAmount   *float64 `gorm:"type:decimal(20,8)"`
	OriginalCurrency *string  `gorm:"type:varchar(3)"`
	ConversionRate   *float64 `gorm:"type:decimal(20,8)"`

	// MoneySource indicates the origin of funds (e.g., Cash, BankAccount, Stripe, etc.)
	MoneySource string `gorm:"type:varchar(64);not null;default:'Internal'"`
}
