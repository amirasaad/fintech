package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"errors"

	"github.com/amirasaad/fintech/pkg/config"
	"github.com/gofiber/fiber/v2"
)

func TestProtected_Unauthorized(t *testing.T) {
	app := fiber.New()
	app.Use(JwtProtected(config.JwtConfig{}))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode == fiber.StatusOK {
		t.Error("expected unauthorized, got 200")
	}
}

func TestJwtError_Malformed(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		return jwtError(c, errors.New("Missing or malformed JWT"))
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected %d, got %d", fiber.StatusBadRequest, resp.StatusCode)
	}
}

func TestJwtError_Invalid(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		return jwtError(c, errors.New("any other error"))
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected %d, got %d", fiber.StatusUnauthorized, resp.StatusCode)
	}
}
