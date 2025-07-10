package webapi

import (
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/middleware"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type NewUser struct {
	Username string `json:"username" validate:"required,max=50,min=3"`
	Email    string `json:"email" validate:"required,email,max=50"`
	Password string `json:"password" validate:"required,min=6,max=72"`
}

type UpdateUserInput struct {
	Names string `json:"names" validate:"max=100"`
}

type PasswordInput struct {
	Password string `json:"password" validate:"required"`
}

func UserRoutes(app *fiber.App, userSvc *service.UserService, authSvc *service.AuthService, cfg *config.AppConfig) {
	app.Get("/user/:id", middleware.JwtProtected(cfg.Jwt), GetUser(userSvc))
	app.Post("/user", CreateUser(userSvc))
	app.Put("/user/:id", middleware.JwtProtected(cfg.Jwt), UpdateUser(userSvc, authSvc))
	app.Delete("/user/:id", middleware.JwtProtected(cfg.Jwt), DeleteUser(userSvc, authSvc))
}

// GetUser retrieves a user by ID.
// @Summary Get user by ID
// @Description Get a user by their unique identifier
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} Response
// @Failure 400 {object} ProblemDetails
// @Failure 404 {object} ProblemDetails
// @Router /user/{id} [get]
// @Security Bearer
func GetUser(userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid user ID: %v", err)
			return ProblemDetailsJSON(c, fiber.StatusBadRequest, "Invalid user ID", err.Error())
		}
		user, err := userSvc.GetUser(id.String())
		if err != nil {
			return ProblemDetailsJSON(c, fiber.StatusNotFound, "No user found with ID", nil)
		}
		return c.JSON(Response{Status: fiber.StatusCreated, Message: "User found", Data: user})
	}
}

// CreateUser creates a new user account.
// @Summary Create a new user
// @Description Create a new user account with username, email, and password
// @Tags users
// @Accept json
// @Produce json
// @Param request body NewUser true "User creation data"
// @Success 201 {object} Response
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Failure 429 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /user [post]
func CreateUser(userSvc *service.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		input, err := BindAndValidate[NewUser](c)
		if err != nil {
			return nil // Error already written by helper
		}
		if len(input.Password) > 72 {
			return ProblemDetailsJSON(c, fiber.StatusBadRequest, "Invalid request body", "Password too long")
		}
		user, err := userSvc.CreateUser(input.Username, input.Email, input.Password)
		if err != nil {
			return ProblemDetailsJSON(c, fiber.StatusInternalServerError, "Couldn't create user", err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(Response{Status: fiber.StatusCreated, Message: "Created user", Data: user})
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
// @Success 200 {object} Response
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Failure 429 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /user/{id} [put]
// @Security Bearer
func UpdateUser(
	userSvc *service.UserService,
	authSvc *service.AuthService,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		input, err := BindAndValidate[UpdateUserInput](c)
		if err != nil {
			return nil // Error already written by helper
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid user ID: %v", err)
			return ProblemDetailsJSON(c, fiber.StatusBadRequest, "Invalid user ID", err.Error())
		}
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return ProblemDetailsJSON(c, fiber.StatusUnauthorized, "unauthorized", "missing user context")
		}
		userID, err := authSvc.GetCurrentUserId(token)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			status := ErrorToStatusCode(err)
			return ProblemDetailsJSON(c, status, "invalid user ID", err.Error())
		}
		if id != userID {
			return ProblemDetailsJSON(c, fiber.StatusForbidden, "You are not allowed to update this user", nil)
		}
		err = userSvc.UpdateUser(id.String(), func(u *user.User) error {
			u.Names = input.Names
			return nil
		})
		if err != nil {
			status := ErrorToStatusCode(err)
			return ProblemDetailsJSON(c, status, "Failed to update user", err.Error())
		}
		// Get the updated user to return in response
		updatedUser, err := userSvc.GetUser(id.String())
		if err != nil {
			return ProblemDetailsJSON(c, fiber.StatusInternalServerError, "Failed to get updated user", err.Error())
		}
		return c.JSON(Response{Status: fiber.StatusOK, Message: "User updated successfully", Data: updatedUser})
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
// @Success 204 {object} Response
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Failure 429 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /user/{id} [delete]
// @Security Bearer
func DeleteUser(
	userSvc *service.UserService,
	authSvc *service.AuthService,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		input, err := BindAndValidate[PasswordInput](c)
		if err != nil {
			return nil // Error already written by helper
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid user ID: %v", err)
			return ProblemDetailsJSON(c, fiber.StatusBadRequest, "Invalid user ID", err.Error())
		}
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return ProblemDetailsJSON(c, fiber.StatusUnauthorized, "unauthorized", "missing user context")
		}
		userID, err := authSvc.GetCurrentUserId(token)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			status := ErrorToStatusCode(err)
			return ProblemDetailsJSON(c, status, "invalid user ID", err.Error())
		}
		if id != userID {
			return ProblemDetailsJSON(c, fiber.StatusForbidden, "You are not allowed to update this user", nil)
		}
		isValid, err := userSvc.ValidUser(id.String(), input.Password)
		if err != nil {
			return ProblemDetailsJSON(c, fiber.StatusInternalServerError, "Failed to validate user", err.Error())
		}
		if !isValid {
			return ProblemDetailsJSON(c, fiber.StatusUnauthorized, "Not valid user", nil)
		}
		err = userSvc.DeleteUser(id.String())
		if err != nil {
			return ProblemDetailsJSON(c, fiber.StatusInternalServerError, "Failed to delete user", err.Error())
		}
		return c.Status(fiber.StatusNoContent).JSON(Response{Status: fiber.StatusNoContent, Message: "User successfully deleted", Data: nil})
	}
}
