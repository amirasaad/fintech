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

		s.Equal(fiber.StatusCreated, resp.StatusCode)
	})

	s.Run("Create account without auth", func() {
		resp := s.MakeRequest("POST", "/account", `{"currency":"USD"}`, "")
		defer resp.Body.Close() //nolint: errcheck
		s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
	})

	s.Run("Create account with invalid currency", func() {
		resp := s.MakeRequest("POST", "/account", `{"currency":"INVALID"}`, s.token)
		defer resp.Body.Close() //nolint: errcheck
		s.Equal(fiber.StatusBadRequest, resp.StatusCode)

		// Verify validation error response format
		var errorResponse common.ProblemDetails
		err := json.NewDecoder(resp.Body).Decode(&errorResponse)
		s.Require().NoError(err)
		s.Equal("Validation failed", errorResponse.Title)
		s.Equal("Request validation failed", errorResponse.Detail)
		s.Equal(fiber.StatusBadRequest, errorResponse.Status)
		s.Equal("about:blank", errorResponse.Type)
		s.NotEmpty(errorResponse.Instance)
		s.NotNil(errorResponse.Errors)
	})
}

func (s *AccountTestSuite) TestDeposit() {
	user := s.CreateTestUser()
	token := s.LoginUser(user)

	depositBody := `{"amount":100,"currency":"USD"}`
	resp := s.MakeRequest("POST", "/account/"+user.ID.String()+"/deposit", depositBody, token)
	// Assert status 202 Accepted
	s.Require().Equal(202, resp.StatusCode)

	// Optionally, check transaction status in DB or via API
	// Optionally, assert event was triggered if MemoryEventBus is accessible
	// TODO: Add event bus assertion when event bus is injectable in E2E
}

func (s *AccountTestSuite) TestWithdraw() {
	// First create an account and deposit some money
	createResp := s.MakeRequest("POST", "/account", `{"currency":"USD"}`, s.token)
	defer createResp.Body.Close() //nolint: errcheck
	s.Equal(fiber.StatusCreated, createResp.StatusCode)

	var createResponse common.Response
	err := json.NewDecoder(createResp.Body).Decode(&createResponse)
	s.Require().NoError(err)

	accountData, ok := createResponse.Data.(map[string]any)
	s.Require().True(ok, "Expected account data to be a map")
	accountID, ok := accountData["ID"].(string)
	s.Require().True(ok, "Expected account ID to be present")

	// Deposit some money first
	depositBody := `{"amount":100,"currency":"USD","money_source":"Cash"}`
	depositResp := s.MakeRequest("POST", fmt.Sprintf("/account/%s/deposit", accountID), depositBody, s.token)
	defer depositResp.Body.Close() //nolint: errcheck
	s.Equal(fiber.StatusOK, depositResp.StatusCode)

	s.Run("Withdraw successfully", func() {
		withdrawBody := `{"amount":50,"currency":"USD","external_target":{"bank_account_number":"1234567890"}}`
		resp := s.MakeRequest("POST", fmt.Sprintf("/account/%s/withdraw", accountID), withdrawBody, s.token)
		defer resp.Body.Close() //nolint: errcheck
		s.Equal(fiber.StatusOK, resp.StatusCode)
	})

	s.Run("Withdraw without auth", func() {
		withdrawBody := `{"amount":50,"currency":"USD","external_target":{"bank_account_number":"1234567890"}}`
		resp := s.MakeRequest("POST", fmt.Sprintf("/account/%s/withdraw", accountID), withdrawBody, "")
		defer resp.Body.Close() //nolint: errcheck
		s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
	})

	s.Run("Withdraw insufficient funds", func() {
		withdrawBody := `{"amount":200,"currency":"USD","external_target":{"bank_account_number":"1234567890"}}`
		resp := s.MakeRequest("POST", fmt.Sprintf("/account/%s/withdraw", accountID), withdrawBody, s.token)
		defer resp.Body.Close() //nolint: errcheck
		s.Equal(fiber.StatusUnprocessableEntity, resp.StatusCode)

		// Verify error response format
		var errorResponse common.ProblemDetails
		err := json.NewDecoder(resp.Body).Decode(&errorResponse)
		s.Require().NoError(err)
		s.Equal("Failed to withdraw", errorResponse.Title)
		s.Equal("insufficient funds for withdrawal", errorResponse.Detail) // Detail should contain the error message
		s.Equal(fiber.StatusUnprocessableEntity, errorResponse.Status)
		s.Equal("about:blank", errorResponse.Type)
		s.NotEmpty(errorResponse.Instance)
	})

	s.Run("Withdraw successfully with external target", func() {
		withdrawBody := `{"amount":50,"currency":"USD","external_target":{"bank_account_number":"1234567890"}}`
		resp := s.MakeRequest("POST", fmt.Sprintf("/account/%s/withdraw", accountID), withdrawBody, s.token)
		defer resp.Body.Close() //nolint: errcheck
		s.Equal(fiber.StatusOK, resp.StatusCode)
	})

	s.Run("Withdraw missing external target", func() {
		withdrawBody := `{"amount":50,"currency":"USD"}`
		resp := s.MakeRequest("POST", fmt.Sprintf("/account/%s/withdraw", accountID), withdrawBody, s.token)
		defer resp.Body.Close() //nolint: errcheck
		s.Equal(fiber.StatusBadRequest, resp.StatusCode)
	})

	s.Run("Withdraw with empty external target", func() {
		withdrawBody := `{"amount":50,"currency":"USD","external_target":{}}`
		resp := s.MakeRequest("POST", fmt.Sprintf("/account/%s/withdraw", accountID), withdrawBody, s.token)
		defer resp.Body.Close() //nolint: errcheck
		s.Equal(fiber.StatusBadRequest, resp.StatusCode)
	})
}

func (s *AccountTestSuite) TestGetBalance() {
	// First create an account
	createResp := s.MakeRequest("POST", "/account", `{"currency":"USD"}`, s.token)
	defer createResp.Body.Close() //nolint: errcheck
	s.Equal(fiber.StatusCreated, createResp.StatusCode)

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
		s.Equal(fiber.StatusOK, resp.StatusCode)
	})

	s.Run("Get balance without auth", func() {
		resp := s.MakeRequest("GET", fmt.Sprintf("/account/%s/balance", accountID), "", "")
		defer resp.Body.Close() //nolint: errcheck
		s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
	})
}
