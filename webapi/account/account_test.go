package account_test

import (
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/webapi/common"
	"github.com/amirasaad/fintech/webapi/testutils"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/suite"
)

type AccountTestSuite struct {
	testutils.E2ETestSuite
	testUser *domain.User
	token    string
}

func (s *AccountTestSuite) SetupTest() {
	// Create test user and login
	s.testUser = s.CreateTestUser()
	s.token = s.LoginUser(s.testUser)
}

func TestAccountTestSuite(t *testing.T) {
	suite.Run(t, new(AccountTestSuite))
}

func (s *AccountTestSuite) TestCreateAccount() {
	s.Run("Create account successfully", func() {
		// Send a valid JSON body for account creation
		resp := s.MakeRequest("POST", "/account", `{"currency":"USD"}`, s.token)
		defer resp.Body.Close() //nolint: errcheck

		// Log the response body for debugging
		body, _ := io.ReadAll(resp.Body)
		s.T().Logf("Response status: %d, body: %s", resp.StatusCode, string(body))

		s.Assert().Equal(fiber.StatusCreated, resp.StatusCode)
	})

	s.Run("Create account without auth", func() {
		resp := s.MakeRequest("POST", "/account", `{"currency":"USD"}`, "")
		defer resp.Body.Close() //nolint: errcheck
		s.Assert().Equal(fiber.StatusUnauthorized, resp.StatusCode)
	})

	s.Run("Create account with invalid currency", func() {
		resp := s.MakeRequest("POST", "/account", `{"currency":"INVALID"}`, s.token)
		defer resp.Body.Close() //nolint: errcheck
		s.Assert().Equal(fiber.StatusBadRequest, resp.StatusCode)

		// Verify validation error response format
		var errorResponse common.ProblemDetails
		err := json.NewDecoder(resp.Body).Decode(&errorResponse)
		s.Require().NoError(err)
		s.Assert().Equal("Validation failed", errorResponse.Title)
		s.Assert().Equal("Request validation failed", errorResponse.Detail)
		s.Assert().Equal(fiber.StatusBadRequest, errorResponse.Status)
		s.Assert().Equal("about:blank", errorResponse.Type)
		s.Assert().NotEmpty(errorResponse.Instance)
		s.Assert().NotNil(errorResponse.Errors)
	})
}

func (s *AccountTestSuite) TestDeposit() {
	// First create an account
	createResp := s.MakeRequest("POST", "/account", `{"currency":"USD"}`, s.token)
	defer createResp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusCreated, createResp.StatusCode)

	// Extract account ID from response
	var createResponse common.Response
	err := json.NewDecoder(createResp.Body).Decode(&createResponse)
	s.Require().NoError(err)

	// The account data is directly in the response data field
	accountData, ok := createResponse.Data.(map[string]any)
	s.Require().True(ok, "Expected account data to be a map")
	accountID, ok := accountData["ID"].(string)
	s.Require().True(ok, "Expected account ID to be present")

	s.Run("Deposit successfully", func() {
		depositBody := `{"amount":100,"currency":"USD"}`
		resp := s.MakeRequest("POST", fmt.Sprintf("/account/%s/deposit", accountID), depositBody, s.token)
		body, _ := io.ReadAll(resp.Body)
		s.T().Logf("Deposit response status: %d, body: %s", resp.StatusCode, string(body))
		defer resp.Body.Close() //nolint: errcheck
		s.Assert().Equal(fiber.StatusOK, resp.StatusCode)
	})

	s.Run("Deposit without auth", func() {
		depositBody := `{"amount":100,"currency":"USD"}`
		resp := s.MakeRequest("POST", fmt.Sprintf("/account/%s/deposit", accountID), depositBody, "")
		defer resp.Body.Close() //nolint: errcheck
		s.Assert().Equal(fiber.StatusUnauthorized, resp.StatusCode)
	})
}

func (s *AccountTestSuite) TestWithdraw() {
	// First create an account and deposit some money
	createResp := s.MakeRequest("POST", "/account", `{"currency":"USD"}`, s.token)
	defer createResp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusCreated, createResp.StatusCode)

	var createResponse common.Response
	err := json.NewDecoder(createResp.Body).Decode(&createResponse)
	s.Require().NoError(err)

	accountData, ok := createResponse.Data.(map[string]any)
	s.Require().True(ok, "Expected account data to be a map")
	accountID, ok := accountData["ID"].(string)
	s.Require().True(ok, "Expected account ID to be present")

	// Deposit some money first
	depositBody := `{"amount":100,"currency":"USD"}`
	depositResp := s.MakeRequest("POST", fmt.Sprintf("/account/%s/deposit", accountID), depositBody, s.token)
	defer depositResp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusOK, depositResp.StatusCode)

	s.Run("Withdraw successfully", func() {
		withdrawBody := `{"amount":50,"currency":"USD"}`
		resp := s.MakeRequest("POST", fmt.Sprintf("/account/%s/withdraw", accountID), withdrawBody, s.token)
		defer resp.Body.Close() //nolint: errcheck
		s.Assert().Equal(fiber.StatusOK, resp.StatusCode)
	})

	s.Run("Withdraw without auth", func() {
		withdrawBody := `{"amount":50,"currency":"USD"}`
		resp := s.MakeRequest("POST", fmt.Sprintf("/account/%s/withdraw", accountID), withdrawBody, "")
		defer resp.Body.Close() //nolint: errcheck
		s.Assert().Equal(fiber.StatusUnauthorized, resp.StatusCode)
	})

	s.Run("Withdraw insufficient funds", func() {
		withdrawBody := `{"amount":200,"currency":"USD"}`
		resp := s.MakeRequest("POST", fmt.Sprintf("/account/%s/withdraw", accountID), withdrawBody, s.token)
		defer resp.Body.Close() //nolint: errcheck
		s.Assert().Equal(fiber.StatusUnprocessableEntity, resp.StatusCode)

		// Verify error response format
		var errorResponse common.ProblemDetails
		err := json.NewDecoder(resp.Body).Decode(&errorResponse)
		s.Require().NoError(err)
		s.Assert().Equal("Failed to withdraw", errorResponse.Title)
		s.Assert().Equal("insufficient funds for withdrawal", errorResponse.Detail) // Detail should contain the error message
		s.Assert().Equal(fiber.StatusUnprocessableEntity, errorResponse.Status)
		s.Assert().Equal("about:blank", errorResponse.Type)
		s.Assert().NotEmpty(errorResponse.Instance)
	})
}

func (s *AccountTestSuite) TestGetBalance() {
	// First create an account
	createResp := s.MakeRequest("POST", "/account", `{"currency":"USD"}`, s.token)
	defer createResp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusCreated, createResp.StatusCode)

	var createResponse common.Response
	err := json.NewDecoder(createResp.Body).Decode(&createResponse)
	s.Require().NoError(err)

	accountData, ok := createResponse.Data.(map[string]any)
	s.Require().True(ok, "Expected account data to be a map")
	accountID, ok := accountData["ID"].(string)
	s.Require().True(ok, "Expected account ID to be present")

	s.Run("Get balance successfully", func() {
		resp := s.MakeRequest("GET", fmt.Sprintf("/account/%s/balance", accountID), "", s.token)
		defer resp.Body.Close() //nolint: errcheck
		s.Assert().Equal(fiber.StatusOK, resp.StatusCode)
	})

	s.Run("Get balance without auth", func() {
		resp := s.MakeRequest("GET", fmt.Sprintf("/account/%s/balance", accountID), "", "")
		defer resp.Body.Close() //nolint: errcheck
		s.Assert().Equal(fiber.StatusUnauthorized, resp.StatusCode)
	})
}
