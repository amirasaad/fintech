package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a user record in the database.
//
//revive:disable
type User struct {
	gorm.Model
	ID                               uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Username                         string    `gorm:"uniqueIndex;not null;size:50" validate:"required,min=3,max=50"`
	Email                            string    `gorm:"uniqueIndex;not null;size:255" validate:"required,email"`
	Password                         string    `gorm:"not null" validate:"required,min=6"`
	Names                            string    `gorm:"size:255"`
	StripeConnectAccountID           string    `gorm:"size:255;index"`
	StripeConnectOnboardingCompleted bool      `gorm:"default:false"`
	StripeConnectAccountStatus       string    `gorm:"size:50"`
	CreatedAt                        time.Time
	UpdatedAt                        time.Time
	DeletedAt                        gorm.DeletedAt `gorm:"index"`
}

//revive:enable

// TableName specifies the table name for the User model.
func (User) TableName() string {
	return "users"
}
