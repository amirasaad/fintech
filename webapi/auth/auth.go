package auth

import (
	authsvc "github.com/amirasaad/fintech/pkg/service/auth"
	"github.com/amirasaad/fintech/webapi/common"
	"github.com/gofiber/fiber/v2"
)

func AuthRoutes(app *fiber.App, authSvc *authsvc.AuthService) {
	app.Post("/auth/login", Login(authSvc))
}

// Login handles user authentication and returns a JWT token.
// @Summary User login
// @Description Authenticate user with identity (username or email) and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginInput true "Login credentials"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.ProblemDetails
// @Failure 401 {object} common.ProblemDetails
// @Failure 429 {object} common.ProblemDetails
// @Failure 500 {object} common.ProblemDetails
// @Router /auth/login [post]
func Login(authSvc *authsvc.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		input, err := common.BindAndValidate[LoginInput](c)
		if input == nil {
			return err // Error already written by BindAndValidate
		}
		user, err := authSvc.Login(c.Context(), input.Identity, input.Password)
		if err != nil {
			// Check if it's an unauthorized error
			if err.Error() == "user unauthorized" {
				return common.ProblemDetailsJSON(c, "Invalid identity or password", nil, "Identity or password is incorrect", fiber.StatusUnauthorized)
			}
			return common.ProblemDetailsJSON(c, "Internal Server Error", err)
		}
		if user == nil {
			return common.ProblemDetailsJSON(c, "Invalid identity or password", nil, "Identity or password is incorrect", fiber.StatusUnauthorized)
		}
		token, err := authSvc.GenerateToken(user)
		if err != nil {
			return common.ProblemDetailsJSON(c, "Internal Server Error", err)
		}
		return common.SuccessResponseJSON(c, fiber.StatusOK, "Success login", fiber.Map{"token": token})
	}
}
