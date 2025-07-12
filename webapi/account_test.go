package webapi

import (
	"encoding/json"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type AccountTestSuite struct {
	E2ETestSuite
	testUser *domain.User
	token    string
}

func (s *AccountTestSuite) SetupTest() {
	// Create test user via POST /user/ endpoint
	s.testUser = s.postToCreateUser()
	s.token = s.loginUser(s.testUser)
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
			// Use the stored token for authenticated requests
			resp := s.makeRequest("POST", "/account", tc.body, s.token)
			defer resp.Body.Close() //nolint:errcheck
			s.Assert().Equal(tc.wantStatus, resp.StatusCode)
		})
	}
}

func (s *AccountTestSuite) TestGetAccountVariants() {
	// Create an account for the success test case
	createResp := s.makeRequest("POST", "/account", `{"currency":"USD"}`, s.token)
	defer createResp.Body.Close() //nolint:errcheck
	s.Assert().Equal(fiber.StatusCreated, createResp.StatusCode)

	var createRespBody Response
	err := json.NewDecoder(createResp.Body).Decode(&createRespBody)
	s.Require().NoError(err)

	// Extract account ID from response
	var accountID string
	if accountData, ok := createRespBody.Data.(map[string]any); ok {
		if id, exists := accountData["ID"]; exists && id != nil {
			accountID = id.(string)
		} else if id, exists := accountData["id"]; exists && id != nil {
			accountID = id.(string)
		}
	} else if account, ok := createRespBody.Data.(*domain.Account); ok {
		accountID = account.ID.String()
	} else {
		s.T().Fatalf("Unexpected response data type: %T", createRespBody.Data)
	}

	s.Require().NotEmpty(accountID, "Account ID should not be empty")

	testCases := []struct {
		desc       string
		accountID  string
		wantStatus int
	}{
		{
			desc:       "account not found",
			accountID:  uuid.New().String(), // Random non-existent account
			wantStatus: fiber.StatusNotFound,
		},
		{
			desc:       "get account balance success",
			accountID:  accountID, // Use the created account
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.desc, func() {
			// Use the stored token for authenticated requests
			resp := s.makeRequest("GET", "/account/"+tc.accountID+"/balance", "", s.token)
			defer resp.Body.Close() //nolint:errcheck
			s.Assert().Equal(tc.wantStatus, resp.StatusCode)
		})
	}
}

func (s *AccountTestSuite) TestDepositVariants() {
	// Create an account for testing
	createResp := s.makeRequest("POST", "/account", `{"currency":"USD"}`, s.token)
	defer createResp.Body.Close() //nolint:errcheck
	s.Assert().Equal(fiber.StatusCreated, createResp.StatusCode)

	var createRespBody Response
	err := json.NewDecoder(createResp.Body).Decode(&createRespBody)
	s.Require().NoError(err)

	// Extract account ID from response
	var accountID string
	if accountData, ok := createRespBody.Data.(map[string]any); ok {
		if id, exists := accountData["ID"]; exists && id != nil {
			accountID = id.(string)
		} else if id, exists := accountData["id"]; exists && id != nil {
			accountID = id.(string)
		}
	} else if account, ok := createRespBody.Data.(*domain.Account); ok {
		accountID = account.ID.String()
	} else {
		s.T().Fatalf("Unexpected response data type: %T", createRespBody.Data)
	}

	s.Require().NotEmpty(accountID, "Account ID should not be empty")

	testCases := []struct {
		desc        string
		body        string
		wantStatus  int
		wantMessage string
		wantConvert bool
	}{
		{
			desc:        "success",
			body:        `{"amount":100.50,"currency":"USD"}`,
			wantStatus:  fiber.StatusOK,
			wantMessage: "Deposit successful",
			wantConvert: false,
		},
		{
			desc:        "success with different currency",
			body:        `{"amount":50.00,"currency":"EUR"}`,
			wantStatus:  fiber.StatusOK,
			wantMessage: "Deposit successful (converted)",
			wantConvert: true,
		},
		{
			desc:       "invalid amount (negative)",
			body:       `{"amount":-10.00,"currency":"USD"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			desc:       "invalid amount (zero)",
			body:       `{"amount":0,"currency":"USD"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			desc:       "missing amount",
			body:       `{"currency":"USD"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			desc:       "invalid amount type",
			body:       `{"amount":"not-a-number","currency":"USD"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			desc:       "invalid currency code",
			body:       `{"amount":100.00,"currency":"INVALID"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			desc:       "malformed JSON",
			body:       `{"amount":100.00,}`,
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.desc, func() {
			resp := s.makeRequest("POST", "/account/"+accountID+"/deposit", tc.body, s.token)
			defer resp.Body.Close() //nolint:errcheck
			s.Assert().Equal(tc.wantStatus, resp.StatusCode)

			if tc.wantStatus == fiber.StatusOK {
				var response Response
				err := json.NewDecoder(resp.Body).Decode(&response)
				s.Require().NoError(err)
				s.Assert().Equal(tc.wantMessage, response.Message)
				s.Assert().NotEmpty(response.Data)
				if tc.wantConvert {
					// Check for conversion fields in response.Data
					if data, ok := response.Data.(map[string]any); ok {
						s.Assert().NotNil(data["original_amount"])
						s.Assert().NotNil(data["converted_amount"])
						s.Assert().NotNil(data["conversion_rate"])
					} else {
						s.T().Fatalf("Expected map[string]any for converted response, got %T", response.Data)
					}
				}
			}
		})
	}
}

func (s *AccountTestSuite) TestWithdrawVariants() {
	// Create an account and deposit some funds for testing
	createResp := s.makeRequest("POST", "/account", `{"currency":"USD"}`, s.token)
	defer createResp.Body.Close() //nolint:errcheck
	s.Assert().Equal(fiber.StatusCreated, createResp.StatusCode)

	var createRespBody Response
	err := json.NewDecoder(createResp.Body).Decode(&createRespBody)
	s.Require().NoError(err)

	// Extract account ID from response
	var accountID string
	if accountData, ok := createRespBody.Data.(map[string]any); ok {
		if id, exists := accountData["ID"]; exists && id != nil {
			accountID = id.(string)
		} else if id, exists := accountData["id"]; exists && id != nil {
			accountID = id.(string)
		}
	} else if account, ok := createRespBody.Data.(*domain.Account); ok {
		accountID = account.ID.String()
	} else {
		s.T().Fatalf("Unexpected response data type: %T", createRespBody.Data)
	}

	s.Require().NotEmpty(accountID, "Account ID should not be empty")

	// Deposit funds first
	depositResp := s.makeRequest("POST", "/account/"+accountID+"/deposit", `{"amount":1000.00,"currency":"USD"}`, s.token)
	defer depositResp.Body.Close() //nolint:errcheck
	s.Assert().Equal(fiber.StatusOK, depositResp.StatusCode)

	testCases := []struct {
		desc        string
		body        string
		wantStatus  int
		wantMessage string
		wantConvert bool
	}{
		{
			desc:        "success",
			body:        `{"amount":100.50,"currency":"USD"}`,
			wantStatus:  fiber.StatusOK,
			wantMessage: "Withdrawal successful",
			wantConvert: false,
		},
		{
			desc:        "success with different currency",
			body:        `{"amount":50.00,"currency":"EUR"}`,
			wantStatus:  fiber.StatusOK,
			wantMessage: "Withdrawal successful (converted)",
			wantConvert: true,
		},
		{
			desc:       "invalid amount (negative)",
			body:       `{"amount":-10.00,"currency":"USD"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			desc:       "invalid amount (zero)",
			body:       `{"amount":0,"currency":"USD"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			desc:       "missing amount",
			body:       `{"currency":"USD"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			desc:       "invalid amount type",
			body:       `{"amount":"not-a-number","currency":"USD"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			desc:       "invalid currency code",
			body:       `{"amount":100.00,"currency":"INVALID"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			desc:       "malformed JSON",
			body:       `{"amount":100.00,}`,
			wantStatus: fiber.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.desc, func() {
			resp := s.makeRequest("POST", "/account/"+accountID+"/withdraw", tc.body, s.token)
			defer resp.Body.Close() //nolint:errcheck
			s.Assert().Equal(tc.wantStatus, resp.StatusCode)

			if tc.wantStatus == fiber.StatusOK {
				var response Response
				err := json.NewDecoder(resp.Body).Decode(&response)
				s.Require().NoError(err)
				s.Assert().Equal(tc.wantMessage, response.Message)
				s.Assert().NotEmpty(response.Data)
				if tc.wantConvert {
					// Check for conversion fields in response.Data
					if data, ok := response.Data.(map[string]any); ok {
						s.Assert().NotNil(data["original_amount"])
						s.Assert().NotNil(data["converted_amount"])
						s.Assert().NotNil(data["conversion_rate"])
					} else {
						s.T().Fatalf("Expected map[string]any for converted response, got %T", response.Data)
					}
				}
			}
		})
	}
}

func (s *AccountTestSuite) TestInsufficientFunds() {
	// Create an account with minimal funds
	createResp := s.makeRequest("POST", "/account", `{"currency":"USD"}`, s.token)
	defer createResp.Body.Close() //nolint:errcheck
	s.Assert().Equal(fiber.StatusCreated, createResp.StatusCode)

	var createRespBody Response
	err := json.NewDecoder(createResp.Body).Decode(&createRespBody)
	s.Require().NoError(err)

	// Extract account ID from response
	var accountID string
	if accountData, ok := createRespBody.Data.(map[string]any); ok {
		if id, exists := accountData["ID"]; exists && id != nil {
			accountID = id.(string)
		} else if id, exists := accountData["id"]; exists && id != nil {
			accountID = id.(string)
		}
	} else if account, ok := createRespBody.Data.(*domain.Account); ok {
		accountID = account.ID.String()
	} else {
		s.T().Fatalf("Unexpected response data type: %T", createRespBody.Data)
	}

	s.Require().NotEmpty(accountID, "Account ID should not be empty")

	// Try to withdraw more than available balance
	resp := s.makeRequest("POST", "/account/"+accountID+"/withdraw", `{"amount":1000.00,"currency":"USD"}`, s.token)
	defer resp.Body.Close() //nolint:errcheck

	// Should get unprocessable entity (422) for insufficient funds
	s.Assert().Equal(fiber.StatusUnprocessableEntity, resp.StatusCode)

	// Validate error response
	var errorResponse ProblemDetails
	err = json.NewDecoder(resp.Body).Decode(&errorResponse)
	s.Require().NoError(err)
	s.Assert().Equal("insufficient funds for withdrawal", errorResponse.Title)
}

func (s *AccountTestSuite) TestAccountNotFound() {
	// Test deposit/withdraw on non-existent account
	nonExistentAccountID := uuid.New().String()

	testCases := []struct {
		desc       string
		endpoint   string
		body       string
		wantStatus int
	}{
		{
			desc:       "deposit to non-existent account",
			endpoint:   "/account/" + nonExistentAccountID + "/deposit",
			body:       `{"amount":100.00,"currency":"USD"}`,
			wantStatus: fiber.StatusNotFound,
		},
		{
			desc:       "withdraw from non-existent account",
			endpoint:   "/account/" + nonExistentAccountID + "/withdraw",
			body:       `{"amount":100.00,"currency":"USD"}`,
			wantStatus: fiber.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.desc, func() {
			resp := s.makeRequest("POST", tc.endpoint, tc.body, s.token)
			defer resp.Body.Close() //nolint:errcheck
			s.Assert().Equal(tc.wantStatus, resp.StatusCode)
		})
	}
}

func (s *AccountTestSuite) TestGetBalanceVariants() {
	// Create an account for the success test case
	createResp := s.makeRequest("POST", "/account", `{"currency":"USD"}`, s.token)
	defer createResp.Body.Close() //nolint:errcheck
	s.Assert().Equal(fiber.StatusCreated, createResp.StatusCode)

	var createRespBody Response
	err := json.NewDecoder(createResp.Body).Decode(&createRespBody)
	s.Require().NoError(err)

	// Extract account ID from response
	var accountID string
	if accountData, ok := createRespBody.Data.(map[string]any); ok {
		if id, exists := accountData["ID"]; exists && id != nil {
			accountID = id.(string)
		} else if id, exists := accountData["id"]; exists && id != nil {
			accountID = id.(string)
		}
	} else if account, ok := createRespBody.Data.(*domain.Account); ok {
		accountID = account.ID.String()
	} else {
		s.T().Fatalf("Unexpected response data type: %T", createRespBody.Data)
	}

	s.Require().NotEmpty(accountID, "Account ID should not be empty")

	testCases := []struct {
		desc        string
		accountID   string
		wantStatus  int
		wantBalance float64
	}{
		{
			desc:       "account not found",
			accountID:  uuid.New().String(), // Random non-existent account
			wantStatus: fiber.StatusNotFound,
		},
		{
			desc:        "get account balance success",
			accountID:   accountID, // Use the created account
			wantStatus:  fiber.StatusOK,
			wantBalance: 0,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.desc, func() {
			resp := s.makeRequest("GET", "/account/"+tc.accountID+"/balance", "", s.token)
			defer resp.Body.Close() //nolint:errcheck
			s.Assert().Equal(tc.wantStatus, resp.StatusCode)
			if tc.wantStatus == fiber.StatusOK {
				var response Response
				err := json.NewDecoder(resp.Body).Decode(&response)
				s.Require().NoError(err)
				// The balance is expected to be in a map structure
				balanceMap, ok := response.Data.(map[string]any)
				s.Require().True(ok, "Expected balance data to be map[string]interface{}, got %T", response.Data)
				balance, ok := balanceMap["balance"].(float64)
				s.Require().True(ok, "Expected balance to be float64, got %T", balanceMap["balance"])
				s.Assert().Equal(tc.wantBalance, balance)
			}
		})
	}
}

func (s *AccountTestSuite) TestGetTransactionsVariants() {
	// Create an account and deposit/withdraw for transaction history
	createResp := s.makeRequest("POST", "/account", `{"currency":"USD"}`, s.token)
	defer createResp.Body.Close() //nolint:errcheck
	s.Assert().Equal(fiber.StatusCreated, createResp.StatusCode)

	var createRespBody Response
	err := json.NewDecoder(createResp.Body).Decode(&createRespBody)
	s.Require().NoError(err)

	// Extract account ID from response
	var accountID string
	if accountData, ok := createRespBody.Data.(map[string]any); ok {
		if id, exists := accountData["ID"]; exists && id != nil {
			accountID = id.(string)
		} else if id, exists := accountData["id"]; exists && id != nil {
			accountID = id.(string)
		}
	} else if account, ok := createRespBody.Data.(*domain.Account); ok {
		accountID = account.ID.String()
	} else {
		s.T().Fatalf("Unexpected response data type: %T", createRespBody.Data)
	}

	s.Require().NotEmpty(accountID, "Account ID should not be empty")

	// Initially, should be empty
	resp := s.makeRequest("GET", "/account/"+accountID+"/transactions", "", s.token)
	defer resp.Body.Close() //nolint:errcheck
	s.Assert().Equal(fiber.StatusOK, resp.StatusCode)
	var response Response
	err = json.NewDecoder(resp.Body).Decode(&response)
	s.Require().NoError(err)
	transactions, ok := response.Data.([]any)
	s.Require().True(ok, "Expected transactions to be []any, got %T", response.Data)
	s.Assert().Len(transactions, 0)

	// Make a deposit
	depositResp := s.makeRequest("POST", "/account/"+accountID+"/deposit", `{"amount":100.00,"currency":"USD"}`, s.token)
	defer depositResp.Body.Close() //nolint:errcheck
	s.Assert().Equal(fiber.StatusOK, depositResp.StatusCode)

	// Make a withdraw
	withdrawResp := s.makeRequest("POST", "/account/"+accountID+"/withdraw", `{"amount":50.00,"currency":"USD"}`, s.token)
	defer withdrawResp.Body.Close() //nolint:errcheck
	s.Assert().Equal(fiber.StatusOK, withdrawResp.StatusCode)

	// Now, should have 2 transactions
	resp2 := s.makeRequest("GET", "/account/"+accountID+"/transactions", "", s.token)
	defer resp2.Body.Close() //nolint:errcheck
	s.Assert().Equal(fiber.StatusOK, resp2.StatusCode)
	var response2 Response
	err = json.NewDecoder(resp2.Body).Decode(&response2)
	s.Require().NoError(err)
	transactions2, ok := response2.Data.([]any)
	s.Require().True(ok, "Expected transactions to be []any, got %T", response2.Data)
	s.Assert().Len(transactions2, 2)

	// Not found case
	nonExistentAccountID := uuid.New().String()
	nfResp := s.makeRequest("GET", "/account/"+nonExistentAccountID+"/transactions", "", s.token)
	defer nfResp.Body.Close() //nolint:errcheck
	s.Assert().Equal(fiber.StatusNotFound, nfResp.StatusCode)
}

func TestAccountTestSuite(t *testing.T) {
	suite.Run(t, new(AccountTestSuite))
}
