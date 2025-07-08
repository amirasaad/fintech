package webapi

import (
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/gofiber/fiber/v2"
)

type LoginInput struct {
	Identity string `json:"identity"`
	Password string `json:"password"`
}

func AuthRoutes(app *fiber.App, authSvc *service.AuthService) {
	app.Post("/login", Login(authSvc))
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
func Login(authSvc *service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		input := new(LoginInput)
		if err := c.BodyParser(input); err != nil {
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Error on login request", err.Error())
		}
		user, err := authSvc.Login(input.Identity, input.Password)
		if err != nil {
			return ErrorResponseJSON(c, fiber.StatusInternalServerError, "Internal Server Error", err.Error())
		}
		if user == nil {
			return ErrorResponseJSON(c, fiber.StatusUnauthorized, "Invalid identity or password", nil)
		}
		token, err := authSvc.GenerateToken(user)
		if err != nil {
			return ErrorResponseJSON(c, fiber.StatusInternalServerError, "Internal Server Error", err.Error())
		}
		return c.JSON(Response{Status: fiber.StatusOK, Message: "Success login", Data: fiber.Map{"token": token}})
	}
}
