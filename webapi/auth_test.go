package webapi

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/suite"
)

type AuthTestSuite struct {
	E2ETestSuite
	testUser *domain.User
}

func (s *AuthTestSuite) SetupTest() {
	// Create test user via POST /user/ endpoint
	s.testUser = s.postToCreateUser()
}

func (s *AuthTestSuite) TestLoginRoute_BadRequest() {
	resp := s.makeRequest("POST", "/auth/login", `{"identity":123}`, "")
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusBadRequest, resp.StatusCode)
}

func (s *AuthTestSuite) TestLoginRoute_Unauthorized() {
	resp := s.makeRequest("POST", "/auth/login", `{"identity":"nonexistent@example.com","password":"password"}`, "")
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *AuthTestSuite) TestLoginRoute_InvalidPassword() {
	loginBody := fmt.Sprintf(`{"identity":"%s","password":"wrongpassword"}`, s.testUser.Email)
	resp := s.makeRequest("POST", "/auth/login", loginBody, "")
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *AuthTestSuite) TestLoginRoute_Success() {
	loginBody := fmt.Sprintf(`{"identity":"%s","password":"password123"}`, s.testUser.Email)
	resp := s.makeRequest("POST", "/auth/login", loginBody, "")
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusOK, resp.StatusCode)

	// Verify response contains token
	var response Response
	err := json.NewDecoder(resp.Body).Decode(&response)
	s.Require().NoError(err)
	loginResponse := response.Data.(map[string]any)
	s.Require().NotEmpty(loginResponse["token"])
}

func (s *AuthTestSuite) TestLoginRoute_InternalServerError() {
	// This test would require mocking the database to simulate an error
	// For now, we'll skip it since we're using a real database
	s.T().Skip("Skipping internal server error test with real database")
}

func (s *AuthTestSuite) TestLoginRoute_ServiceError() {
	// This test would require mocking the database to simulate an error
	// For now, we'll skip it since we're using a real database
	s.T().Skip("Skipping service error test with real database")
}

func TestAuthTestSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}
