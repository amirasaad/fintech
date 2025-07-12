package webapi

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type UserTestSuite struct {
	E2ETestSuiteWithDB
	testUser *domain.User
	token    string
}

func (s *UserTestSuite) SetupTest() {
	// Create test user via POST /user/ endpoint
	s.testUser = s.postToCreateUser()
	s.token = s.loginUser(s.testUser)
}

func (s *UserTestSuite) TestCreateUserVariants() {
	testCases := []struct {
		desc       string
		body       string
		wantStatus int
	}{
		{
			desc:       "success",
			body:       `{"username":"newuser","email":"new@example.com","password":"password123"}`,
			wantStatus: fiber.StatusCreated,
		},
		{
			desc:       "invalid body",
			body:       `{"username":123}`,
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.desc, func() {
			resp := s.makeRequest("POST", "/user", tc.body, "")
			defer resp.Body.Close() //nolint:errcheck
			s.Assert().Equal(tc.wantStatus, resp.StatusCode)
		})
	}
}

func (s *UserTestSuite) TestGetUserVariants() {
	testCases := []struct {
		userId     string
		desc       string
		wantStatus int
	}{
		{
			userId:     uuid.New().String(),
			desc:       "user not found",
			wantStatus: fiber.StatusUnauthorized, // Generic error for security
		},
		{
			userId:     s.testUser.ID.String(),
			desc:       "get user success",
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.desc, func() {

			resp := s.makeRequest("GET", "/user/"+tc.userId, "", s.token)
			defer resp.Body.Close() //nolint:errcheck
			s.Assert().Equal(tc.wantStatus, resp.StatusCode)
		})
	}
}

func (s *UserTestSuite) TestUpdateUserVariants() {
	testCases := []struct {
		desc       string
		body       string
		wantStatus int
	}{
		{
			desc:       "success",
			body:       `{"names":"newname"}`,
			wantStatus: fiber.StatusOK,
		},
		{
			desc:       "invalid body",
			body:       `{"names":123}`,
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.desc, func() {
			resp := s.makeRequest("PUT", "/user/"+s.testUser.ID.String(), tc.body, s.token)
			defer resp.Body.Close() //nolint:errcheck
			s.Assert().Equal(tc.wantStatus, resp.StatusCode)
		})
	}
}

func (s *UserTestSuite) TestDeleteUserVariants() {
	// NOTE: Test isolation issue with mock expectations
	// The 'invalid_password' and 'internal_error' tests fail when run in the full suite
	// due to mock expectation bleeding between tests, but work correctly in isolation.
	// This is a known limitation of the test suite setup that doesn't affect actual functionality.
	// The API behavior is correct: invalid credentials return 401, internal errors return 500.

	testCases := []struct {
		desc       string
		body       string
		wantStatus int
	}{
		{
			desc:       "success",
			body:       `{"password":"password123"}`,
			wantStatus: fiber.StatusNoContent,
		},
		{
			desc:       "invalid body",
			body:       `{"pass":123}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			desc:       "invalid password",
			body:       `{"password":"wrongpass"}`,
			wantStatus: fiber.StatusUnauthorized, // This is correct - invalid credentials should return 401
		},
	}

	for _, tc := range testCases {
		s.Run(tc.desc, func() {
			// Skip failing tests in CI due to test isolation issue
			// if tc.desc == "invalid password" || tc.desc == "internal error" {
			// 	s.T().Skip("Skipping due to test isolation issue with mock expectations in test suite")
			// }

			// Create a fresh user for each test case to avoid conflicts when user is deleted
			testUser := s.postToCreateUser()
			token := s.loginUser(testUser)

			// Generate a real JWT token for authenticated requests
			resp := s.makeRequest("DELETE", "/user/"+testUser.ID.String(), tc.body, token)
			defer resp.Body.Close() //nolint:errcheck
			s.Assert().Equal(tc.wantStatus, resp.StatusCode)
		})
	}
}

func TestUserTestSuite(t *testing.T) {
	suite.Run(t, new(UserTestSuite))
}
