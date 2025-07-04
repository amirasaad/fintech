package webapi

import (
	"github.com/amirasaad/fintech/pkg/middleware"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func UserRoutes(app *fiber.App, uowFactory func() (repository.UnitOfWork, error)) {
	app.Get("/user/:id", middleware.Protected(), GetUser(uowFactory))
	app.Post("/user", CreateUser(uowFactory))
	app.Put("/user/:id", middleware.Protected(), UpdateUser(uowFactory))
	app.Delete("/user/:id", middleware.Protected(), DeleteUser(uowFactory))
}

// GetUser get a user
func GetUser(uowFactory func() (repository.UnitOfWork, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for deposit: %v", err)
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Invalid user ID", err.Error())
		}
		userService := service.NewUserService(uowFactory)
		user, err := userService.GetUser(id)
		if err != nil {
			return ErrorResponseJSON(c, fiber.StatusNotFound, "No user found with ID", nil)
		}
		return c.JSON(Response{Status: fiber.StatusCreated, Message: "User found", Data: user})
	}
}

// CreateUser new user
func CreateUser(uowFactory func() (repository.UnitOfWork, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		type NewUser struct {
			Username string `json:"username" validate:"required,max=50"`
			Email    string `json:"email" validate:"required,email,max=50"`
			Password string `json:"password" validate:"required,min=6"`
		}

		var newUser NewUser
		if err := c.BodyParser(&newUser); err != nil {
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Review your input", err.Error())
		}
		validate := validator.New()
		if err := validate.Struct(newUser); err != nil {
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Invalid request body", err.Error())
		}
		if len(newUser.Password) > 72 {
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Invalid request body", "Password too long")
		}
		service := service.NewUserService(uowFactory)
		user, err := service.CreateUser(newUser.Username, newUser.Email, newUser.Password)
		if err != nil {
			return ErrorResponseJSON(c, fiber.StatusInternalServerError, "Couldn't create user", err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(Response{Status: fiber.StatusCreated, Message: "Created user", Data: user})
	}
}

// UpdateUser update user
func UpdateUser(uowFactory func() (repository.UnitOfWork, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		type UpdateUserInput struct {
			Names string `json:"names"`
		}
		var uui UpdateUserInput
		if err := c.BodyParser(&uui); err != nil {
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Review your input", err.Error())
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for deposit: %v", err)
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Invalid user ID", err.Error())
		}
		authSvc := service.NewAuthService(uowFactory)
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return ErrorResponseJSON(c, fiber.StatusUnauthorized, "unauthorized", "missing user context")
		}
		userID, err := authSvc.GetCurrentUserId(token)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			status := ErrorToStatusCode(err)
			return ErrorResponseJSON(c, status, "invalid user ID", err.Error())
		}
		if id != userID {
			return ErrorResponseJSON(c, fiber.StatusForbidden, "You are not allowed to update this user", nil)
		}

		userService := service.NewUserService(uowFactory)
		user, err := userService.GetUser(id)
		if err != nil {
			return ErrorResponseJSON(c, fiber.StatusNotFound, "User not found", nil)
		}
		err = userService.UpdateUser(user)
		if err != nil {
			return ErrorResponseJSON(c, fiber.StatusInternalServerError, "Failed to update user", err.Error())
		}
		return c.JSON(Response{Status: fiber.StatusOK, Message: "User updated successfully", Data: user})
	}
}

// DeleteUser delete user
func DeleteUser(uowFactory func() (repository.UnitOfWork, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		type PasswordInput struct {
			Password string `json:"password"`
		}
		var pi PasswordInput
		if err := c.BodyParser(&pi); err != nil {
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Review your input", err.Error())
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for deposit: %v", err)
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Invalid user ID", err.Error())
		}
		authSvc := service.NewAuthService(uowFactory)
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return ErrorResponseJSON(c, fiber.StatusUnauthorized, "unauthorized", "missing user context")
		}
		userID, err := authSvc.GetCurrentUserId(token)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			status := ErrorToStatusCode(err)
			return ErrorResponseJSON(c, status, "invalid user ID", err.Error())
		}
		if id != userID {
			return ErrorResponseJSON(c, fiber.StatusForbidden, "You are not allowed to update this user", nil)
		}
		userService := service.NewUserService(uowFactory)
		if !userService.ValidUser(id, pi.Password) {
			return ErrorResponseJSON(c, fiber.StatusUnauthorized, "Not valid user", nil)
		}

		err = userService.DeleteUser(id)
		if err != nil {
			return ErrorResponseJSON(c, fiber.StatusInternalServerError, "Failed to delete user", err.Error())
		}
		return c.Status(fiber.StatusNoContent).JSON(Response{Status: fiber.StatusNoContent, Message: "User successfully deleted", Data: nil})
	}
}
