package webapi

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/domain"

	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/suite"
)

type RateLimitTestSuite struct {
	suite.Suite
	app *fiber.App
}

func (s *RateLimitTestSuite) SetupTest() {
	// Provide dummy services for required arguments
	dummyUow := func() (repository.UnitOfWork, error) { return nil, nil }
	accountSvc := service.NewAccountService(dummyUow, domain.NewStubCurrencyConverter())
	userSvc := service.NewUserService(dummyUow)
	authSvc := &service.AuthService{} // Use zero value or a mock if available

	s.app = NewApp(accountSvc, userSvc, authSvc, config.AppConfig{})
}

func (s *RateLimitTestSuite) TestRateLimit() {
	s.T().Parallel()
	// Send requests until rate limit is hit
	for i := range [6]int{} { // Default limit is 5 requests per IP per second
		req := httptest.NewRequest(fiber.MethodGet, "/", nil)
		resp, err := s.app.Test(req, 1000) // Add timeout to app.Test
		s.Require().NoError(err)
		defer resp.Body.Close() //nolint: errcheck

		if i < 5 {
			s.Assert().Equal(fiber.StatusOK, resp.StatusCode, "Expected OK for request %d", i+1)
		} else {
			s.Assert().Equal(fiber.StatusTooManyRequests, resp.StatusCode, "Expected Too Many Requests for request %d", i+1)
		}
	}

	// Wait for the rate limit window to reset
	time.Sleep(1 * time.Second)

	// Send another request and expect it to be successful
	req := httptest.NewRequest(fiber.MethodGet, "/", nil)
	resp, err := s.app.Test(req, 1000) // Add timeout to app.Test
	s.Require().NoError(err)
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusOK, resp.StatusCode, "Expected OK after rate limit reset")
}

func TestRateLimitTestSuite(t *testing.T) {
	suite.Run(t, new(RateLimitTestSuite))
}
