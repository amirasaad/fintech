package common

import (
	"testing"
	"time"

	infra_provider "github.com/amirasaad/fintech/infra/provider"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/currency"

	"log/slog"

	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service/account"
	"github.com/amirasaad/fintech/pkg/service/auth"
	currencyservice "github.com/amirasaad/fintech/pkg/service/currency"
	"github.com/amirasaad/fintech/pkg/service/user"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/suite"
)

type RateLimitTestSuite struct {
	suite.Suite
	app *fiber.App
}

func (s *RateLimitTestSuite) SetupTest() {
	// Provide dummy services for required arguments
	dummyUow := *new(repository.UnitOfWork)
	accountSvc := account.NewAccountService(dummyUow, infra_provider.NewStubCurrencyConverter(), slog.Default())
	userSvc := user.NewUserService(dummyUow, slog.Default())

	// Create a dummy auth strategy and service
	dummyAuthStrategy := auth.NewJWTAuthStrategy(dummyUow, config.JwtConfig{}, slog.Default())
	authSvc := auth.NewAuthService(dummyUow, dummyAuthStrategy, slog.Default())

	// Create a dummy currency registry and service
	dummyRegistry := &currency.CurrencyRegistry{}
	currencySvc := currencyservice.NewCurrencyService(dummyRegistry, slog.Default())

	s.app = NewApp(accountSvc, userSvc, authSvc, currencySvc, &config.AppConfig{})
}

func (s *RateLimitTestSuite) TestRateLimit() {
	s.T().Parallel()
	// Send requests until rate limit is hit
	for i := range [6]int{} { // Default limit is 5 requests per IP per second
		resp := MakeRequestWithApp(s.app, fiber.MethodGet, "/", "", "")
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
	resp := MakeRequestWithApp(s.app, fiber.MethodGet, "/", "", "")
	defer resp.Body.Close() //nolint: errcheck
	s.Assert().Equal(fiber.StatusOK, resp.StatusCode, "Expected OK after rate limit reset")
}

func TestRateLimitTestSuite(t *testing.T) {
	suite.Run(t, new(RateLimitTestSuite))
}
