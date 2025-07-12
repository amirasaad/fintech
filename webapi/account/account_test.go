package account

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/amirasaad/fintech/pkg/apiutil"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/testutils"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/suite"
)

type AccountTestSuite struct {
	suite.Suite
	app      *fiber.App
	testUser *domain.User
	token    string
}

func (s *AccountTestSuite) SetupTest() {
	// Setup test app with real database
	app, _, testUser, _, _ := testutils.SetupTestAppWithTestcontainers(s.T())
	s.app = app
	s.testUser = testUser
	s.token = testutils.LoginUser(app, testUser)
}

func TestAccountTestSuite(t *testing.T) {
	suite.Run(t, new(AccountTestSuite))
}

func (s *AccountTestSuite) TestCreateAccount() {
	s.Run("Create account successfully", func() {
		resp := testutils.MakeRequest(s.app, "POST", "/account", "", s.token)
		defer resp.Body.Close() //nolint: errcheck
		s.Assert().Equal(fiber.StatusCreated, resp.StatusCode)
	})

	s.Run("Create account without auth", func() {
		resp := testutils.MakeRequest(s.app, "POST", "/account", "", "")
		defer resp.Body.Close() //nolint: errcheck
		s.Assert().Equal(fiber.StatusUnauthorized, resp.StatusCode)
	})
}

func (s *AccountTestSuite) TestDeposit() {
	// First create an account
	createResp := testutils.MakeRequest(s.app, "POST", "/account", "", s.token)
	defer createResp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusCreated, createResp.StatusCode)

	// Extract account ID from response
	var createResponse apiutil.Response
	err := json.NewDecoder(createResp.Body).Decode(&createResponse)
	s.Require().NoError(err)

	accountData, ok := createResponse.Data.(map[string]any)
	s.Require().True(ok)
	accountID, ok := accountData["id"].(string)
	s.Require().True(ok)

	s.Run("Deposit successfully", func() {
		depositBody := fmt.Sprintf(`{"amount":100,"currency":"USD"}`)
		resp := testutils.MakeRequest(s.app, "POST", fmt.Sprintf("/account/%s/deposit", accountID), depositBody, s.token)
		defer resp.Body.Close() //nolint: errcheck
		s.Assert().Equal(fiber.StatusOK, resp.StatusCode)
	})

	s.Run("Deposit without auth", func() {
		depositBody := fmt.Sprintf(`{"amount":100,"currency":"USD"}`)
		resp := testutils.MakeRequest(s.app, "POST", fmt.Sprintf("/account/%s/deposit", accountID), depositBody, "")
		defer resp.Body.Close() //nolint: errcheck
		s.Assert().Equal(fiber.StatusUnauthorized, resp.StatusCode)
	})
}

func (s *AccountTestSuite) TestWithdraw() {
	// First create an account and deposit some money
	createResp := testutils.MakeRequest(s.app, "POST", "/account", "", s.token)
	defer createResp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusCreated, createResp.StatusCode)

	var createResponse apiutil.Response
	err := json.NewDecoder(createResp.Body).Decode(&createResponse)
	s.Require().NoError(err)

	accountData, ok := createResponse.Data.(map[string]any)
	s.Require().True(ok)
	accountID, ok := accountData["id"].(string)
	s.Require().True(ok)

	// Deposit some money first
	depositBody := fmt.Sprintf(`{"amount":100,"currency":"USD"}`)
	depositResp := testutils.MakeRequest(s.app, "POST", fmt.Sprintf("/account/%s/deposit", accountID), depositBody, s.token)
	defer depositResp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusOK, depositResp.StatusCode)

	s.Run("Withdraw successfully", func() {
		withdrawBody := fmt.Sprintf(`{"amount":50,"currency":"USD"}`)
		resp := testutils.MakeRequest(s.app, "POST", fmt.Sprintf("/account/%s/withdraw", accountID), withdrawBody, s.token)
		defer resp.Body.Close() //nolint: errcheck
		s.Assert().Equal(fiber.StatusOK, resp.StatusCode)
	})

	s.Run("Withdraw without auth", func() {
		withdrawBody := fmt.Sprintf(`{"amount":50,"currency":"USD"}`)
		resp := testutils.MakeRequest(s.app, "POST", fmt.Sprintf("/account/%s/withdraw", accountID), withdrawBody, "")
		defer resp.Body.Close() //nolint: errcheck
		s.Assert().Equal(fiber.StatusUnauthorized, resp.StatusCode)
	})
}

func (s *AccountTestSuite) TestGetBalance() {
	// First create an account
	createResp := testutils.MakeRequest(s.app, "POST", "/account", "", s.token)
	defer createResp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusCreated, createResp.StatusCode)

	var createResponse apiutil.Response
	err := json.NewDecoder(createResp.Body).Decode(&createResponse)
	s.Require().NoError(err)

	accountData, ok := createResponse.Data.(map[string]any)
	s.Require().True(ok)
	accountID, ok := accountData["id"].(string)
	s.Require().True(ok)

	s.Run("Get balance successfully", func() {
		resp := testutils.MakeRequest(s.app, "GET", fmt.Sprintf("/account/%s/balance", accountID), "", s.token)
		defer resp.Body.Close() //nolint: errcheck
		s.Assert().Equal(fiber.StatusOK, resp.StatusCode)
	})

	s.Run("Get balance without auth", func() {
		resp := testutils.MakeRequest(s.app, "GET", fmt.Sprintf("/account/%s/balance", accountID), "", "")
		defer resp.Body.Close() //nolint: errcheck
		s.Assert().Equal(fiber.StatusUnauthorized, resp.StatusCode)
	})
}
