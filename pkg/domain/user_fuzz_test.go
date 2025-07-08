package domain_test

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
)

// FuzzNewUser tests NewUser invariants with random input.
func FuzzNewUser(f *testing.F) {
	f.Add("testuser", "test@example.com", "password123") // Seed input
	f.Add("", "", "")
	f.Fuzz(func(t *testing.T, username, email, password string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("NewUser panicked: %v (username=%q, email=%q, password=%q)", r, username, email, password)
			}
		}()
		user, err := domain.NewUser(username, email, password)
		if err == nil {
			if user.Username == "" || user.Email == "" {
				t.Errorf("User has empty username or email: username=%q, email=%q", user.Username, user.Email)
			}
		}
	})
}
