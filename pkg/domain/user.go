package domain

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Password string    `json:"password"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}
func NewUser(username, email, password string) (*User, error) {
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	return &User{
		ID:       uuid.New(),
		Username: username,
		Email:    email,
		Password: hashedPassword,
		Created:  time.Now().UTC(),
		Updated:  time.Now().UTC(),
	}, nil
}

func NewUserFromData(id uuid.UUID, username, email, password string, created, updated time.Time) *User {
	return &User{
		ID:       id,
		Username: username,
		Email:    email,
		Password: password,
		Created:  created,
		Updated:  updated,
	}
}
