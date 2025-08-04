package user

import (
	"errors"
	"time"

	"github.com/amirasaad/fintech/pkg/utils"
	"github.com/google/uuid"
)

var (
	// ErrUserNotFound is returned when a user cannot be found in the
	// repository.
	ErrUserNotFound = errors.New("user not found")
	// ErrUserUnauthorized is return when user
	ErrUserUnauthorized = errors.New("user unauthorized")
)

// User represents a user in the system.
type User struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	Names     string    `json:"names"`
	CreatedAt time.Time `json:"created"`
	UpdatedAt time.Time `json:"updated"`
}

// NewUser creates a new User with a hashed password and current timestamps.
func NewUser(username, email, password string) (*User, error) {
	if username == "" {
		return nil, errors.New("username cannot be empty")
	}
	if email == "" {
		return nil, errors.New("email cannot be empty")
	}
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}
	return &User{
		ID:        uuid.New(),
		Username:  username,
		Email:     email,
		Password:  hashedPassword,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}, nil
}

// NewUserFromData creates a User from raw data (used for DB hydration).
func NewUserFromData(
	id uuid.UUID,
	username, email, password string,
	created, updated time.Time,
) *User {
	return &User{
		ID:        id,
		Username:  username,
		Email:     email,
		Password:  password,
		CreatedAt: created,
		UpdatedAt: updated,
	}
}
