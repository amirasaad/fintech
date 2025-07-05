package webapi

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestRateLimit(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	app := NewApp(nil, nil) // Pass nil for uowFactory as it's not needed for this test

	// Send requests until rate limit is hit
	for i := range [6]int{} { // Default limit is 5 requests per IP per second
		req := httptest.NewRequest(fiber.MethodGet, "/", nil)
		resp, err := app.Test(req, 1000) // Add timeout to app.Test
		assert.NoError(err)
		defer resp.Body.Close() //nolint:errcheck

		if i < 5 {
			assert.Equal(fiber.StatusOK, resp.StatusCode, "Expected OK for request %d", i+1)
		} else {
			assert.Equal(fiber.StatusTooManyRequests, resp.StatusCode, "Expected Too Many Requests for request %d", i+1)
		}
	}

	// Wait for the rate limit window to reset
	time.Sleep(1 * time.Second)

	// Send another request and expect it to be successful
	req := httptest.NewRequest(fiber.MethodGet, "/", nil)
	resp, err := app.Test(req, 1000) // Add timeout to app.Test
	assert.NoError(err)
	defer resp.Body.Close() //nolint:errcheck
	assert.Equal(fiber.StatusOK, resp.StatusCode, "Expected OK after rate limit reset")
}
