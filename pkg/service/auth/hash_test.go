package auth

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/utils"
)

func TestHardcodedHash(t *testing.T) {
	// This is the hardcoded hash from BasicAuthStrategy that matches the password "password"
	const hardcodedHash = "$2a$10$.IIxpSc3OElWXLV2Wj517eUGmZ64IQgBNQ4OcFbanW85CTrgrIDQy"

	// Test if the hash is for the password "password"
	if !utils.CheckPasswordHash("password", hardcodedHash) {
		t.Errorf("Hardcoded hash does not match password 'password'")
	}

	// Test with a different password to ensure the test is working
	if utils.CheckPasswordHash("wrongpassword", hardcodedHash) {
		t.Error("Expected error for wrong password, but got none")
	}
}
