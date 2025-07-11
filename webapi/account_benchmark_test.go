package webapi

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type AccountBenchmarkTestSuite struct {
	E2ETestSuiteWithDB
}

func (s *AccountBenchmarkTestSuite) SetupTest() {
	// Create test user in database
	s.createTestUserInDB()
}

func (s *AccountBenchmarkTestSuite) BenchmarkAccountCreate(b *testing.B) {
	// Generate a real JWT token for authenticated requests
	token := s.generateTestToken()

	body := `{"currency":"USD"}`
	req := httptest.NewRequest("POST", "/account", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := s.app.Test(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close() //nolint:errcheck
	}
}

func (s *AccountBenchmarkTestSuite) BenchmarkAccountDeposit(b *testing.B) {
	// Generate a real JWT token for authenticated requests
	token := s.generateTestToken()

	accountID := uuid.New()

	body := `{"amount":100.0}`
	req := httptest.NewRequest("POST", "/account/"+accountID.String()+"/deposit", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := s.app.Test(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close() //nolint:errcheck
	}
}

func TestAccountBenchmarkTestSuite(t *testing.T) {
	suite.Run(t, new(AccountBenchmarkTestSuite))
}
