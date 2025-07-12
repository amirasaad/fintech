package webapi

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type AccountBenchmarkTestSuite struct {
	E2ETestSuite
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

	b.ResetTimer()
	for b.Loop() {
		resp := s.makeRequest("POST", "/account", body, token)
		resp.Body.Close() //nolint:errcheck
	}
}

func (s *AccountBenchmarkTestSuite) BenchmarkAccountDeposit(b *testing.B) {

	accountID := uuid.New()

	body := `{"amount":100.0}`

	b.ResetTimer()
	for b.Loop() {
		resp := s.makeRequest("POST", "/account/"+accountID.String()+"/deposit", body, s.token)
		resp.Body.Close() //nolint:errcheck
	}
}

func TestAccountBenchmarkTestSuite(t *testing.T) {
	suite.Run(t, new(AccountBenchmarkTestSuite))
}
