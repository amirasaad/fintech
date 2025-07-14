package webapi_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	infra_provider "github.com/amirasaad/fintech/infra/provider"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service/account"
	"github.com/amirasaad/fintech/pkg/service/auth"
	currencyservice "github.com/amirasaad/fintech/pkg/service/currency"
	userservice "github.com/amirasaad/fintech/pkg/service/user"
	"github.com/amirasaad/fintech/webapi"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestRateLimit(t *testing.T) {
	// Create app with stricter rate limits for testing
	cfg := &config.AppConfig{
		RateLimit: config.RateLimitConfig{
			MaxRequests: 5,
			Window:      1 * time.Second,
		},
	}

	// Provide dummy services for required arguments
	dummyUow := repository.UnitOfWork(nil)
	accountSvc := account.NewService(config.Deps{
		Uow:             dummyUow,
		Converter:       infra_provider.NewStubCurrencyConverter(),
		Logger:          slog.Default(),
		PaymentProvider: infra_provider.NewMockPaymentProvider(),
	})
	userSvc := userservice.NewUserService(config.Deps{
		Uow: dummyUow, Logger: slog.Default(),
	})

	// Create a dummy auth strategy and service
	dummyAuthStrategy := auth.NewJWTAuthStrategy(dummyUow, config.JwtConfig{}, slog.Default())
	authSvc := auth.NewAuthService(dummyUow, dummyAuthStrategy, slog.Default())

	// Create a dummy currency registry and service
	dummyRegistry := &currency.CurrencyRegistry{}
	currencySvc := currencyservice.NewCurrencyService(dummyRegistry, slog.Default())

	app := webapi.NewApp(accountSvc, userSvc, authSvc, currencySvc, cfg)

	// Helper function to make requests
	makeRequest := func(method, path, body, token string) *http.Response {
		var req *http.Request
		if body != "" {
			req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req = httptest.NewRequest(method, path, nil)
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := app.Test(req)
		if err != nil {
			panic(err)
		}
		return resp
	}

	// Send requests until rate limit is hit
	for i := 0; i < 6; i++ { // Test with 5 requests per second limit
		resp := makeRequest(fiber.MethodGet, "/", "", "")
		defer resp.Body.Close() //nolint: errcheck

		if i < 5 {
			assert.Equal(t, fiber.StatusOK, resp.StatusCode, "Expected OK for request %d", i+1)
		} else {
			assert.Equal(t, fiber.StatusTooManyRequests, resp.StatusCode, "Expected Too Many Requests for request %d", i+1)
		}
	}

	// Wait for the rate limit window to reset
	time.Sleep(1 * time.Second)

	// Send another request and expect it to be successful
	resp := makeRequest(fiber.MethodGet, "/", "", "")
	defer resp.Body.Close() //nolint: errcheck
	assert.Equal(t, fiber.StatusOK, resp.StatusCode, "Expected OK after rate limit reset")
}
