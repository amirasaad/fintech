package webapi

import (
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type AccountTestSuite struct {
	E2ETestSuiteWithDB
}

func (s *AccountTestSuite) SetupTest() {
	// Create test user in database
	s.createTestUserInDB()
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
			// Generate a real JWT token for authenticated requests
			token := s.generateTestToken()
			resp := s.makeRequest("POST", "/account", tc.body, token)
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
			wantStatus: fiber.StatusUnauthorized, // Generic error for security
		},
		{
			desc:       "get account success",
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.desc, func() {
			id := uuid.New()
			// Generate a real JWT token for authenticated requests
			token := s.generateTestToken()
			resp := s.makeRequest("GET", "/account/"+id.String(), "", token)
			defer resp.Body.Close() //nolint:errcheck
			s.Assert().Equal(tc.wantStatus, resp.StatusCode)
		})
	}
}

func TestAccountTestSuite(t *testing.T) {
	suite.Run(t, new(AccountTestSuite))
}
