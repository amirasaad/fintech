package webapi

import (
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/gofiber/fiber/v2"
)

func AuthRoutes(app *fiber.App, uowFactory func() (repository.UnitOfWork, error), strategy service.AuthStrategy) {
	app.Post("/login", Login(uowFactory, strategy))
}

// Login get user and password
func Login(uowFactory func() (repository.UnitOfWork, error), strategy service.AuthStrategy) fiber.Handler {
	return func(c *fiber.Ctx) error {
		type LoginInput struct {
			Identity string `json:"identity"`
			Password string `json:"password"`
		}
		input := new(LoginInput)

		if err := c.BodyParser(input); err != nil {
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Error on login request", err.Error())
		}

		authService := service.NewAuthService(uowFactory, strategy)
		user, token, err := authService.Login(input.Identity, input.Password)
		if err != nil {
			return ErrorResponseJSON(c, fiber.StatusInternalServerError, "Internal Server Error", err.Error())
		}
		if user == nil || token == "" {
			return ErrorResponseJSON(c, fiber.StatusUnauthorized, "Invalid identity or password", nil)
		}
		return c.JSON(Response{Status: fiber.StatusOK, Message: "Success login", Data: fiber.Map{"token": token}})
	}
}
