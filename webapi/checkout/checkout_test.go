package checkout_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/amirasaad/fintech/webapi/testutils"
	"github.com/stretchr/testify/suite"
)

type CheckoutE2ETestSuite struct {
	testutils.E2ETestSuite
}

func (s *CheckoutE2ETestSuite) TestGetPendingSessionsE2E() {
	user := s.CreateTestUser()
	token := s.LoginUser(user)

	resp := s.MakeRequest(
		"GET",
		"/checkout/sessions/pending",
		"",
		token,
	)

	s.Require().Equal(http.StatusOK, resp.StatusCode)

	var responseBody map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&responseBody)
	s.Require().NoError(err)

	data, ok := responseBody["data"].([]interface{})
	s.Require().True(ok, "Expected 'data' to be an array")
	s.Require().Empty(data, "Expected no pending sessions initially")
}

func TestCheckoutE2ETestSuite(t *testing.T) {
	suite.Run(t, new(CheckoutE2ETestSuite))
}
