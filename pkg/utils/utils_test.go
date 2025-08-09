package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword(t *testing.T) {
	password := "testpassword"
	hashedPassword, err := HashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hashedPassword)
	assert.NotEqual(t, password, hashedPassword)
}

func TestCheckPasswordHash(t *testing.T) {
	password := "testpassword"
	hashedPassword, _ := HashPassword(password)

	// Test with correct password
	assert.True(t, CheckPasswordHash(password, hashedPassword))

	// Test with incorrect password
	assert.False(t, CheckPasswordHash("wrongpassword", hashedPassword))
}

func TestIsEmail(t *testing.T) {
	// Test with valid emails
	assert.True(t, IsEmail("test@example.com"))
	assert.True(t, IsEmail("another.test@sub.domain.co.uk"))

	// Test with invalid emails
	assert.False(t, IsEmail("invalid-email"))
	assert.False(t, IsEmail("invalid@.com"))
	assert.False(t, IsEmail("@example.com"))
	// assert.False(t, IsEmail("test@example"))
}
