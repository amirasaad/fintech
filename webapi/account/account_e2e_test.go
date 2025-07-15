package account_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/amirasaad/fintech/webapi/common"
	"github.com/amirasaad/fintech/webapi/testutils"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/suite"
)

type AccountE2ETestSuite struct {
	testutils.E2ETestSuite
}

func TestAccountE2ETestSuite(t *testing.T) {
	suite.Run(t, new(AccountE2ETestSuite))
}

func (s *AccountE2ETestSuite) TestDepositEndToEnd() {
	// 1. Register user
	userEmail := fmt.Sprintf("e2euser_%d@example.com", time.Now().UnixNano())
	registerBody := fmt.Sprintf(`{"email":"%s","password":"password123"}`, userEmail)
	resp := s.MakeRequest("POST", "/auth/register", registerBody, "")
	defer resp.Body.Close() //nolint:errcheck
	s.Equal(fiber.StatusOK, resp.StatusCode)

	// 2. Login
	loginBody := fmt.Sprintf(`{"email":"%s","password":"password123"}`, userEmail)
	resp = s.MakeRequest("POST", "/auth/login", loginBody, "")
	defer resp.Body.Close() //nolint:errcheck
	s.Equal(fiber.StatusOK, resp.StatusCode)
	var loginResp common.Response
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&loginResp))
	loginData := loginResp.Data.(map[string]any)
	token := loginData["token"].(string)

	// 3. Create account
	createAccountBody := `{"currency":"USD"}`
	resp = s.MakeRequest("POST", "/account", createAccountBody, token)
	defer resp.Body.Close() //nolint:errcheck
	s.Equal(fiber.StatusCreated, resp.StatusCode)
	var createAccountResp common.Response
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&createAccountResp))
	accountData := createAccountResp.Data.(map[string]any)
	accountID := accountData["ID"].(string)

	// 4. Deposit
	depositBody := `{"amount":100,"currency":"USD","money_source":"Cash"}`
	resp = s.MakeRequest("POST", fmt.Sprintf("/account/%s/deposit", accountID), depositBody, token)
	defer resp.Body.Close() //nolint:errcheck
	s.Equal(fiber.StatusAccepted, resp.StatusCode)

	// 5. Wait for async processing
	time.Sleep(5 * time.Second)

	// 6. Check balance
	resp = s.MakeRequest("GET", fmt.Sprintf("/account/%s/balance", accountID), "", token)
	defer resp.Body.Close() //nolint:errcheck
	s.Equal(fiber.StatusOK, resp.StatusCode)
	var balanceResp common.Response
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&balanceResp))
	balanceData := balanceResp.Data.(map[string]any)
	balance := balanceData["balance"].(float64)
	s.InDelta(100.0, balance, 0.01)

	// 7. Check transactions
	resp = s.MakeRequest("GET", fmt.Sprintf("/account/%s/transactions", accountID), "", token)
	defer resp.Body.Close() //nolint:errcheck
	s.Equal(fiber.StatusOK, resp.StatusCode)
	var txResp common.Response
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&txResp))
	txs, ok := txResp.Data.([]any)
	s.Require().True(ok)
	s.NotEmpty(txs)
	// Check at least one transaction is completed
	foundCompleted := false
	for _, t := range txs {
		tx := t.(map[string]any)
		if tx["status"] == "completed" {
			foundCompleted = true
		}
	}
	s.True(foundCompleted, "At least one transaction should be completed after deposit")
}
