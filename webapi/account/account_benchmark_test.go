package account_test

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/webapi/testutils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type AccountBenchmarkTestSuite struct {
	testutils.E2ETestSuite
	testUser *domain.User
	token    string
}

func (s *AccountBenchmarkTestSuite) SetupTest() {
	// Create test user via POST /user/ endpoint
	s.testUser = s.CreateTestUser()
	s.token = s.LoginUser(s.testUser)
}

func (s *AccountBenchmarkTestSuite) BenchmarkAccountCreate(b *testing.B) {
	// Generate a real JWT token for authenticated requests
	token := s.LoginUser(s.testUser)

	body := `{"currency":"USD"}`

	b.ResetTimer()
	for b.Loop() {
		resp := s.MakeRequest("POST", "/account", body, token)
		resp.Body.Close() //nolint:errcheck
	}
}

func (s *AccountBenchmarkTestSuite) BenchmarkAccountDeposit(b *testing.B) {

	accountID := uuid.New()

	body := `{"amount":100.0}`

	b.ResetTimer()
	for b.Loop() {
		resp := s.MakeRequest("POST", "/account/"+accountID.String()+"/deposit", body, s.token)
		resp.Body.Close() //nolint:errcheck
	}
}

func TestAccountBenchmarkTestSuite(t *testing.T) {
	suite.Run(t, new(AccountBenchmarkTestSuite))
}
