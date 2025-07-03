package middleware

import (
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestProtectedMiddleware(t *testing.T) {
	app := fiber.New()
	app.Use(Protected())
	// TODO: Add request and assert response
	t.Log("middleware test placeholder")
}
