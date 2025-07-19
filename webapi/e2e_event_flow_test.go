package webapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/amirasaad/fintech/webapi/testutils"
	"github.com/stretchr/testify/suite"
)

type E2EFlowsTestSuite struct {
	testutils.E2ETestSuite
}

func (s *E2EFlowsTestSuite) TestDepositE2E() {
	user := s.CreateTestUser()
	token := s.LoginUser(user)

	// Create account for user
	accountBody := fmt.Sprintf(`{"user_id":"%s","currency":"USD"}`, user.ID)
	resp := s.MakeRequest("POST", "/account", accountBody, token)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	var accountResp map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&accountResp)
	accountID := accountResp["data"].(map[string]interface{})["id"].(string)

	// Make deposit request
	depositBody := `{"amount":100,"currency":"USD"}`
	resp = s.MakeRequest("POST", fmt.Sprintf("/account/%s/deposit", accountID), depositBody, token)
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	var depositResp map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&depositResp)
	s.T().Logf("Deposit response: %+v", depositResp)
}

func (s *E2EFlowsTestSuite) TestWithdrawE2E() {
	user := s.CreateTestUser()
	token := s.LoginUser(user)

	// Create account for user
	accountBody := fmt.Sprintf(`{"user_id":"%s","currency":"USD"}`, user.ID)
	resp := s.MakeRequest("POST", "/account", accountBody, token)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	var accountResp map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&accountResp)
	accountID := accountResp["data"].(map[string]interface{})["id"].(string)

	// Deposit first to ensure sufficient balance
	depositBody := `{"amount":200,"currency":"USD"}`
	resp = s.MakeRequest("POST", fmt.Sprintf("/account/%s/deposit", accountID), depositBody, token)
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	// Make withdraw request
	withdrawBody := `{"amount":100,"currency":"USD","external_target":{"bank_account_number":"123456789","routing_number":"987654321"}}`
	resp = s.MakeRequest("POST", fmt.Sprintf("/account/%s/withdraw", accountID), withdrawBody, token)
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	var withdrawResp map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&withdrawResp)
	s.T().Logf("Withdraw response: %+v", withdrawResp)
}

func (s *E2EFlowsTestSuite) TestTransferE2E() {
	user := s.CreateTestUser()
	token := s.LoginUser(user)

	// Create two accounts for user
	accountBody := fmt.Sprintf(`{"user_id":"%s","currency":"USD"}`, user.ID)
	resp := s.MakeRequest("POST", "/account", accountBody, token)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	var accountResp1 map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&accountResp1)
	accountID1 := accountResp1["data"].(map[string]interface{})["id"].(string)

	resp = s.MakeRequest("POST", "/account", accountBody, token)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	var accountResp2 map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&accountResp2)
	accountID2 := accountResp2["data"].(map[string]interface{})["id"].(string)

	// Deposit to first account
	depositBody := `{"amount":150,"currency":"USD"}`
	resp = s.MakeRequest("POST", fmt.Sprintf("/account/%s/deposit", accountID1), depositBody, token)
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	// Make transfer request
	transferBody := fmt.Sprintf(`{"amount":100,"currency":"USD","target_account_id":"%s"}`, accountID2)
	resp = s.MakeRequest("POST", fmt.Sprintf("/account/%s/transfer", accountID1), transferBody, token)
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	var transferResp map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&transferResp)
	s.T().Logf("Transfer response: %+v", transferResp)
}

func TestE2EFlowsTestSuite(t *testing.T) {
	suite.Run(t, new(E2EFlowsTestSuite))
}
