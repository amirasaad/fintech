package transaction

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Transaction represents a persisted financial transaction.
type Transaction struct {
	gorm.Model
	ID        uuid.UUID `gorm:"type:uuid;primary_key"`
	AccountID uuid.UUID `gorm:"type:uuid"`
	UserID    uuid.UUID `gorm:"type:uuid"`
	Amount    int64
	Currency  string `gorm:"type:varchar(3);not null;default:'USD'"`
	Balance   int64
	Status    string  `gorm:"type:varchar(32);not null;default:'pending'"`
	PaymentID *string `gorm:"type:varchar(64);column:payment_id;index"`

	// Conversion fields (nullable when no conversion occurs)
	OriginalAmount   *float64 `gorm:"type:decimal(20,8)"`
	OriginalCurrency *string  `gorm:"type:varchar(3)"`
	ConversionRate   *float64 `gorm:"type:decimal(20,8)"`

	// MoneySource indicates the origin of funds (e.g., Cash, BankAccount, Stripe, etc.)
	MoneySource          string `gorm:"type:varchar(64);not null;default:'Internal'"`
	ExternalTargetMasked string `gorm:"type:varchar(128);column:external_target_masked"`

	// TargetCurrency is the currency the account is credited in (for multi-currency deposits)
	TargetCurrency string `gorm:"type:varchar(8);column:target_currency"`

	// Fee is the transaction fee in the smallest currency unit (e.g., cents)
	Fee *int64 `gorm:"type:bigint;default:0"`
}

// TableName specifies the table name for the Transaction model.
func (Transaction) TableName() string {
	return "transactions"
}
