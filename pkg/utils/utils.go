package utils

import (
	"net/mail"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a plain password using bcrypt with cost 14.
func HashPassword(password string) (string, error) {
	return hashPassword(password)
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// CheckPasswordHash compares a plain password with a bcrypt hash.
func CheckPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// IsEmail returns true if the string is a valid email address.
func IsEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
