package dto

import (
	"time"

	"github.com/google/uuid"
)

// UserCreate represents the data needed to create a new user.
type UserCreate struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username" validate:"required,min=3,max=50"`
	Email    string    `json:"email" validate:"required,email"`
	Password string    `json:"password,omitempty" validate:"required,min=6"`
	Names    string    `json:"names,omitempty"`
}

// UserUpdate represents the data that can be updated for a user.
type UserUpdate struct {
	Username *string `json:"username,omitempty" validate:"omitempty,min=3,max=50"`
	Email    *string `json:"email,omitempty" validate:"omitempty,email"`
	Password *string `json:"password,omitempty" validate:"omitempty,min=6"`
	Names    *string `json:"names,omitempty"`
}

// UserRead represents a read-optimized view of a user.
type UserRead struct {
	ID             uuid.UUID `json:"id"`
	Username       string    `json:"username"`
	HashedPassword string    `json:"hashed_password"`
	Email          string    `json:"email"`
	Names          string    `json:"names,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
