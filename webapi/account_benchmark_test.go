package webapi

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type AccountBenchmarkTestSuite struct {
	E2ETestSuiteWithDB
	testUser *domain.User
	token    string
}

func (s *AccountBenchmarkTestSuite) SetupTest() {
	// Create test user via POST /user/ endpoint
	s.testUser = s.postToCreateUser()
	s.token = s.loginUser(s.testUser)
}

func (s *AccountBenchmarkTestSuite) BenchmarkAccountCreate(b *testing.B) {
	// Generate a real JWT token for authenticated requests
	token := s.loginUser(s.testUser)

	body := `{"currency":"USD"}`
	req := httptest.NewRequest("POST", "/account", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	b.ResetTimer()
	for b.Loop() {
		resp, err := s.app.Test(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close() //nolint:errcheck
	}
}

func (s *AccountBenchmarkTestSuite) BenchmarkAccountDeposit(b *testing.B) {

	accountID := uuid.New()

	body := `{"amount":100.0}`
	req := httptest.NewRequest("POST", "/account/"+accountID.String()+"/deposit", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.token)

	b.ResetTimer()
	for b.Loop() {
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
