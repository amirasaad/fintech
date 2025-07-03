package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestProtected_Unauthorized(t *testing.T) {
	app := fiber.New()
	app.Use(Protected())
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode == fiber.StatusOK {
		t.Error("expected unauthorized, got 200")
	}
}
