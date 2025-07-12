package user

import (
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/middleware"
	authsvc "github.com/amirasaad/fintech/pkg/service/auth"
	usersvc "github.com/amirasaad/fintech/pkg/service/user"
	"github.com/amirasaad/fintech/webapi/common"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func UserRoutes(app *fiber.App, userSvc *usersvc.UserService, authSvc *authsvc.AuthService, cfg *config.AppConfig) {
	app.Get("/user/:id", middleware.JwtProtected(cfg.Jwt), GetUser(userSvc))
	app.Post("/user", CreateUser(userSvc))
	app.Put("/user/:id", middleware.JwtProtected(cfg.Jwt), UpdateUser(userSvc, authSvc))
	app.Delete("/user/:id", middleware.JwtProtected(cfg.Jwt), DeleteUser(userSvc, authSvc))
}

// GetUser returns a Fiber handler for retrieving a user by ID.
// @Summary Get user by ID
// @Description Retrieve a user by their ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.ProblemDetails
// @Failure 401 {object} common.ProblemDetails
// @Failure 404 {object} common.ProblemDetails
// @Router /user/{id} [get]
// @Security Bearer
func GetUser(userSvc *usersvc.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid user ID: %v", err)
			return common.ProblemDetailsJSON(c, "Invalid user ID", err, "User ID must be a valid UUID", fiber.StatusBadRequest)
		}
		user, err := userSvc.GetUser(c.Context(), id.String())
		if err != nil || user == nil {
			// Generic error for not found to prevent user enumeration
			return common.ProblemDetailsJSON(c, "Invalid credentials", nil, fiber.StatusUnauthorized)
		}
		return common.SuccessResponseJSON(c, fiber.StatusOK, "User found", user)
	}
}

// CreateUser creates a new user account.
// @Summary Create a new user
// @Description Create a new user account with username, email, and password
// @Tags users
// @Accept json
// @Produce json
// @Param request body NewUser true "User creation data"
// @Success 201 {object} common.Response
// @Failure 400 {object} common.ProblemDetails
// @Failure 401 {object} common.ProblemDetails
// @Failure 429 {object} common.ProblemDetails
// @Failure 500 {object} common.ProblemDetails
// @Router /user [post]
func CreateUser(userSvc *usersvc.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		input, err := common.BindAndValidate[NewUser](c)
		if input == nil {
			return err // error response already written
		}
		if len(input.Password) > 72 {
			return common.ProblemDetailsJSON(c, "Invalid request body", nil, "Password too long")
		}
		user, err := userSvc.CreateUser(c.Context(), input.Username, input.Email, input.Password)
		if err != nil {
			return common.ProblemDetailsJSON(c, "Couldn't create user", err)
		}
		return common.SuccessResponseJSON(c, fiber.StatusCreated, "Created user", user)
	}
}

// UpdateUser updates user information.
// @Summary Update user
// @Description Update user information by ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body UpdateUserInput true "User update data"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.ProblemDetails
// @Failure 401 {object} common.ProblemDetails
// @Failure 429 {object} common.ProblemDetails
// @Failure 500 {object} common.ProblemDetails
// @Router /user/{id} [put]
// @Security Bearer
func UpdateUser(
	userSvc *usersvc.UserService,
	authSvc *authsvc.AuthService,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		input, err := common.BindAndValidate[UpdateUserInput](c)
		if input == nil {
			return err // error response already written
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid user ID: %v", err)
			return common.ProblemDetailsJSON(c, "Invalid user ID", err, "User ID must be a valid UUID", fiber.StatusBadRequest)
		}
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return common.ProblemDetailsJSON(c, "Unauthorized", nil, "missing user context", fiber.StatusUnauthorized)
		}
		userID, err := authSvc.GetCurrentUserId(token)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			return common.ProblemDetailsJSON(c, "Unauthorized", nil, fiber.StatusUnauthorized)
		}
		if id != userID {
			return common.ProblemDetailsJSON(c, "Forbidden", nil, "You are not allowed to update this user", fiber.StatusUnauthorized)
		}
		err = userSvc.UpdateUser(c.Context(), id.String(), func(u *user.User) error {
			u.Names = input.Names
			return nil
		})
		if err != nil {
			// Generic error for update failure
			return common.ProblemDetailsJSON(c, "Invalid credentials", nil, fiber.StatusUnauthorized)
		}
		// Get the updated user to return in response
		updatedUser, err := userSvc.GetUser(c.Context(), id.String())
		if err != nil || updatedUser == nil {
			return common.ProblemDetailsJSON(c, "Invalid credentials", nil, fiber.StatusUnauthorized)
		}
		return common.SuccessResponseJSON(c, fiber.StatusOK, "User updated successfully", updatedUser)
	}
}

// DeleteUser deletes a user account.
// @Summary Delete user
// @Description Delete a user account by ID with password confirmation
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body PasswordInput true "Password confirmation"
// @Success 204 {object} common.Response
// @Failure 400 {object} common.ProblemDetails
// @Failure 401 {object} common.ProblemDetails
// @Failure 429 {object} common.ProblemDetails
// @Failure 500 {object} common.ProblemDetails
// @Router /user/{id} [delete]
// @Security Bearer
func DeleteUser(
	userSvc *usersvc.UserService,
	authSvc *authsvc.AuthService,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		input, err := common.BindAndValidate[PasswordInput](c)
		if input == nil {
			return err // error response already written
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid user ID: %v", err)
			return common.ProblemDetailsJSON(c, "Invalid user ID", err, "User ID must be a valid UUID", fiber.StatusBadRequest)
		}
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return common.ProblemDetailsJSON(c, "Unauthorized", nil, "missing user context", fiber.StatusUnauthorized)
		}
		userID, err := authSvc.GetCurrentUserId(token)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			return common.ProblemDetailsJSON(c, "Unauthorized", nil, fiber.StatusUnauthorized)
		}
		if id != userID {
			return common.ProblemDetailsJSON(c, "Forbidden", nil, "You are not allowed to update this user", fiber.StatusUnauthorized)
		}
		isValid, err := userSvc.ValidUser(c.Context(), id.String(), input.Password)
		if err != nil {
			// If this is a DB/internal error, return 500
			return common.ProblemDetailsJSON(c, "Failed to validate user", err, fiber.StatusInternalServerError)
		}
		if !isValid {
			// Invalid password or user not found
			return common.ProblemDetailsJSON(c, "Invalid credentials", nil, fiber.StatusUnauthorized)
		}
		err = userSvc.DeleteUser(c.Context(), id.String())
		if err != nil {
			return common.ProblemDetailsJSON(c, "Failed to delete user", err, fiber.StatusInternalServerError)
		}
		return common.SuccessResponseJSON(c, fiber.StatusNoContent, "User successfully deleted", nil)
	}
}
