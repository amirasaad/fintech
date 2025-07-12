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
func TestAccountTestSuite(t *testing.T) {
	suite.Run(t, new(AccountTestSuite))
}
