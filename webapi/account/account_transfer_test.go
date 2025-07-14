package account_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/webapi/common"
	"github.com/amirasaad/fintech/webapi/testutils"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/suite"
)

type AccountTransferTestSuite struct {
	testutils.E2ETestSuite
	testUser *domain.User
	token    string
}

func (s *AccountTransferTestSuite) SetupTest() {
	s.testUser = s.CreateTestUser()
	s.token = s.LoginUser(s.testUser)
}

func (s *AccountTransferTestSuite) TestTransfer_Success() {
	// Create source account
	sourceResp := s.MakeRequest("POST", "/account", `{"currency":"USD"}`, s.token)
	defer sourceResp.Body.Close() //nolint:errcheck
	s.Equal(fiber.StatusCreated, sourceResp.StatusCode)
	var sourceAccountResp common.Response
	s.Require().NoError(json.NewDecoder(sourceResp.Body).Decode(&sourceAccountResp))
	sourceData := sourceAccountResp.Data.(map[string]any)
	sourceAccountID := sourceData["ID"].(string)

	// Create destination account
	destResp := s.MakeRequest("POST", "/account", `{"currency":"USD"}`, s.token)
	defer destResp.Body.Close() //nolint:errcheck
	s.Equal(fiber.StatusCreated, destResp.StatusCode)
	var destAccountResp common.Response
	s.Require().NoError(json.NewDecoder(destResp.Body).Decode(&destAccountResp))
	destData := destAccountResp.Data.(map[string]any)
	destAccountID := destData["ID"].(string)

	// Deposit funds into source account
	depositBody := `{"amount":100,"currency":"USD","money_source":"Cash"}`
	depositResp := s.MakeRequest("POST", fmt.Sprintf("/account/%s/deposit", sourceAccountID), depositBody, s.token)
	defer depositResp.Body.Close() //nolint:errcheck
	s.Equal(fiber.StatusOK, depositResp.StatusCode)

	// Attempt transfer
	transferBody := fmt.Sprintf(`{"amount":50,"currency":"USD","destination_account_id":"%s"}`, destAccountID)
	transferResp := s.MakeRequest("POST", fmt.Sprintf("/account/%s/transfer", sourceAccountID), transferBody, s.token)
	defer transferResp.Body.Close() //nolint:errcheck

	s.Equal(fiber.StatusOK, transferResp.StatusCode, "Transfer endpoint should return 200 OK")

	// Parse and validate response structure
	var transferResponse common.Response
	s.Require().NoError(json.NewDecoder(transferResp.Body).Decode(&transferResponse))
	data, ok := transferResponse.Data.(map[string]any)
	s.Require().True(ok, "Expected response data to be a map")
	// Check outgoing and incoming transactions
	_, ok = data["outgoing_transaction"]
	s.True(ok, "Outgoing transaction should be present in response")
	_, ok = data["incoming_transaction"]
	s.True(ok, "Incoming transaction should be present in response")
}

func TestTransferE2ETestSuite(t *testing.T) {
	suite.Run(t, new(AccountTransferTestSuite))
}
