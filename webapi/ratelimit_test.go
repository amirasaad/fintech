package webapi_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/amirasaad/fintech/infra/eventbus"
	infra_provider "github.com/amirasaad/fintech/infra/provider"
	"github.com/amirasaad/fintech/pkg/app"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/webapi"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestRateLimit(t *testing.T) {
	// Create app with stricter rate limits for testing
	cfg := &config.App{
		RateLimit: &config.RateLimit{},
		Auth: &config.Auth{
			Jwt: &config.Jwt{},
		},
	}

	// Provide dummy services for required arguments
	dummyUow := repository.UnitOfWork(nil)

	app := webapi.SetupApp(app.New(&app.Deps{
		Uow:                  dummyUow,
		EventBus:             eventbus.NewWithMemory(slog.Default()),
		ExchangeRateProvider: infra_provider.NewMockExchangeRate(),
		PaymentProvider:      infra_provider.NewMockPaymentProvider(),
		Logger:               slog.Default(),
	}, cfg))

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
	for i := 0; i < cfg.RateLimit.MaxRequests; i++ {
		resp := makeRequest(fiber.MethodGet, "/", "", "")
		defer resp.Body.Close() //nolint: errcheck

		if i < cfg.RateLimit.MaxRequests+1 {
			assert.Equal(t, fiber.StatusOK, resp.StatusCode, "Expected OK for request %d", i+1)
		} else {
			assert.Equal(
				t,
				fiber.StatusTooManyRequests,
				resp.StatusCode,
				"Expected Too Many Requests for request %d",
				i+1,
			)
		}
	}

	// Wait for the rate limit window to reset
	time.Sleep(1 * time.Second)

	// Send another request and expect it to be successful
	resp := makeRequest(fiber.MethodGet, "/", "", "")
	defer resp.Body.Close() //nolint: errcheck
	assert.Equal(t, fiber.StatusOK, resp.StatusCode, "Expected OK after rate limit reset")
}
