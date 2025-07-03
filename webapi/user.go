package webapi

import (
	"github.com/amirasaad/fintech/pkg/middleware"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
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
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		uow, err := uowFactory()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to create unit of work", "data": nil})
		}

		user, err := uow.UserRepository().Get(id)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"status": "error", "message": "No user found with ID", "data": nil})
		}

		return c.JSON(fiber.Map{"status": "success", "message": "User found", "data": user})
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
		return c.JSON(fiber.Map{"status": "success", "message": "Created user", "data": user})
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
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Review your input", "errors": err.Error()})
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for deposit: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		userID, err := GetCurrentUserId(c)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			return ErrorResponseJSON(c, fiber.StatusUnauthorized, "invalid user ID", nil)
		}
		if id != userID {
			return ErrorResponseJSON(c, fiber.StatusForbidden, "You are not allowed to update this user", nil)
		}

		service := service.NewUserService(uowFactory)
		user, err := service.GetUser(id)
		if err != nil {
			return ErrorResponseJSON(c, fiber.StatusNotFound, "User not found", nil)
		}
		err = service.UpdateUser(user)
		if err != nil {
			return ErrorResponseJSON(c, fiber.StatusInternalServerError, "Failed to update user", err.Error())
		}
		return c.JSON(fiber.Map{"status": "success", "message": "User updated successfully", "data": user})
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
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Review your input", "errors": err.Error()})
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for deposit: %v", err)
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Invalid account ID", err.Error())
		}
		userID, err := GetCurrentUserId(c)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			return ErrorResponseJSON(c, fiber.StatusUnauthorized, "invalid user ID", nil)
		}
		if id != userID {
			return ErrorResponseJSON(c, fiber.StatusForbidden, "You are not allowed to update this user", nil)
		}
		service := service.NewUserService(uowFactory)
		if !service.ValidUser(id, pi.Password) {
			return ErrorResponseJSON(c, fiber.StatusInternalServerError, "Not valid user", nil)
		}

		err = service.DeleteUser(id, pi.Password)
		if err != nil {
			return ErrorResponseJSON(c, fiber.StatusInternalServerError, "Failed to delete user", err.Error())
		}
		return c.JSON(fiber.Map{"status": "success", "message": "User successfully deleted", "data": nil})
	}
}
