package webapi_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
	data, ok := accountResp["data"].(map[string]interface{})
	s.Require().True(ok, "Expected accountResp['data'] to be a map, got: %#v", accountResp["data"])
	accountID, ok := data["ID"].(string)
	if !ok {
		// Log the full response for debugging
		b, _ := json.MarshalIndent(accountResp, "", "  ")
		s.T().Logf("Full account creation response: %s", string(b))
	}
	s.Require().True(ok, "Expected data['ID'] to be a string, got: %#v", data["ID"])

	// Make deposit request
	depositBody := `{"amount":100,"currency":"USD", "money_source": "cash"}`
	resp = s.MakeRequest("POST", fmt.Sprintf("/account/%s/deposit", accountID), depositBody, token)
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		s.T().Logf("Deposit response body: %s", string(b))
	}
	s.Require().Equal(http.StatusAccepted, resp.StatusCode)
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
	data, ok := accountResp["data"].(map[string]interface{})
	s.Require().True(ok, "Expected accountResp['data'] to be a map, got: %#v", accountResp["data"])
	accountID, ok := data["ID"].(string)
	s.Require().True(ok, "Expected data['ID'] to be a string, got: %#v", data["ID"])

	// Deposit first to ensure sufficient balance
	depositBody := `{"amount":200,"currency":"USD", "money_source": "cash"}`
	resp = s.MakeRequest("POST", fmt.Sprintf("/account/%s/deposit", accountID), depositBody, token)
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		s.T().Logf("Deposit response body: %s", string(b))
	}
	s.Require().Equal(http.StatusAccepted, resp.StatusCode)

	// Make withdraw request
	withdrawBody := `{"amount":100,"currency":"USD","external_target":{"bank_account_number":"123456789","routing_number":"987654321"}, "money_source": "cash"}`
	resp = s.MakeRequest("POST", fmt.Sprintf("/account/%s/withdraw", accountID), withdrawBody, token)
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		s.T().Logf("Withdraw response body: %s", string(b))
	}
	s.Require().Equal(http.StatusAccepted, resp.StatusCode)
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
	data, ok := accountResp1["data"].(map[string]interface{})
	s.Require().True(ok, "Expected accountResp1['data'] to be a map, got: %#v", accountResp1["data"])
	accountID1, ok := data["ID"].(string)
	if !ok {
		b, _ := json.MarshalIndent(accountResp1, "", "  ")
		s.T().Logf("Full account creation response 1: %s", string(b))
	}
	s.Require().True(ok, "Expected data1['ID'] to be a string, got: %#v", data["ID"])

	resp = s.MakeRequest("POST", "/account", accountBody, token)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	var accountResp2 map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&accountResp2)
	data, ok = accountResp2["data"].(map[string]interface{})
	s.Require().True(ok, "Expected accountResp2['data'] to be a map, got: %#v", accountResp2["data"])
	accountID2, ok := data["ID"].(string)
	if !ok {
		b, _ := json.MarshalIndent(accountResp2, "", "  ")
		s.T().Logf("Full account creation response 2: %s", string(b))
	}
	s.Require().True(ok, "Expected data2['ID'] to be a string, got: %#v", data["ID"])

	// Deposit to first account
	depositBody := `{"amount":150,"currency":"USD", "money_source": "cash"}`
	resp = s.MakeRequest("POST", fmt.Sprintf("/account/%s/deposit", accountID1), depositBody, token)
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		s.T().Logf("Deposit response body: %s", string(b))
	}
	s.Require().Equal(http.StatusAccepted, resp.StatusCode)

	// Make transfer request
	transferBody := fmt.Sprintf(`{"amount":100,"currency":"USD","destination_account_id":"%s", "money_source": "cash"}`, accountID2)
	resp = s.MakeRequest("POST", fmt.Sprintf("/account/%s/transfer", accountID1), transferBody, token)
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		s.T().Logf("Transfer response body: %s", string(b))
	}
	s.Require().Equal(http.StatusAccepted, resp.StatusCode)
	var transferResp map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&transferResp)
	s.T().Logf("Transfer response: %+v", transferResp)
}

func TestE2EFlowsTestSuite(t *testing.T) {
	if os.Getenv("E2E") == "" {
		t.Skip("Skipping E2E tests")
	}
	suite.Run(t, new(testutils.E2ETestSuite))
}
