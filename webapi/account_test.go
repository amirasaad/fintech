package webapi

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type AccountTestSuite struct {
	E2ETestSuite
	testUser *domain.User
	token    string
}

func (s *AccountTestSuite) SetupTest() {
	// Create test user via POST /user/ endpoint
	s.testUser = s.postToCreateUser()
	s.token = s.loginUser(s.testUser)
}

func (s *AccountTestSuite) TestCreateAccountVariants() {
	testCases := []struct {
		desc       string
		body       string
		wantStatus int
	}{
		{
			desc:       "success",
			body:       `{"currency":"USD"}`,
			wantStatus: fiber.StatusCreated,
		},
		{
			desc:       "invalid body",
			body:       `{"currency":123}`,
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.desc, func() {
			// Use the stored token for authenticated requests
			resp := s.makeRequest("POST", "/account", tc.body, s.token)
			defer resp.Body.Close() //nolint:errcheck
			s.Assert().Equal(tc.wantStatus, resp.StatusCode)
		})
	}
}

func (s *AccountTestSuite) TestGetAccountVariants() {
	testCases := []struct {
		desc       string
		wantStatus int
	}{
		{
			desc:       "account not found",
			wantStatus: fiber.StatusNotFound, // Account doesn't exist
		},
		{
			desc:       "get account balance success",
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.desc, func() {
			id := uuid.New()
			// Use the stored token for authenticated requests
			resp := s.makeRequest("GET", "/account/"+id.String()+"/balance", "", s.token)
			defer resp.Body.Close() //nolint:errcheck
			s.Assert().Equal(tc.wantStatus, resp.StatusCode)
		})
	}
}

func TestAccountTestSuite(t *testing.T) {
	suite.Run(t, new(AccountTestSuite))
}
