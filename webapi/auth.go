package webapi

import (
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/gofiber/fiber/v2"
)

type LoginInput struct {
	Identity string `json:"identity"`
	Password string `json:"password"`
}

func AuthRoutes(app *fiber.App, uowFactory func() (repository.UnitOfWork, error), strategy service.AuthStrategy) {
	app.Post("/login", Login(uowFactory, strategy))
}

// Login handles user authentication and returns a JWT token.
// @Summary User login
// @Description Authenticate user with identity (username or email) and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginInput true "Login credentials"
// @Success 200 {object} Response
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Failure 429 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /login [post]
func Login(uowFactory func() (repository.UnitOfWork, error), strategy service.AuthStrategy) fiber.Handler {
	return func(c *fiber.Ctx) error {

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
