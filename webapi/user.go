package webapi

import (
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

func validToken(t *jwt.Token, id uuid.UUID) bool {
	claims := t.Claims.(jwt.MapClaims)
	uid := claims["user_id"].(uuid.UUID)

	return uid == id
}
func UserRoutes(app *fiber.App, uowFactory func() (repository.UnitOfWork, error)) {
	app.Get("/user/:id", GetUser(uowFactory))
	app.Post("/user", CreateUser(uowFactory))
	app.Put("/user/:id", UpdateUser(uowFactory))
	app.Delete("/user/:id", DeleteUser(uowFactory))
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
		defer uow.Rollback()

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
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Review your input", "errors": err.Error()})
		}
		validate := validator.New()
		if err := validate.Struct(newUser); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body", "errors": err.Error()})
		}
		if len(newUser.Password) > 72 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body", "errors": "Password too long"})
		}
		service := service.NewUserService(uowFactory)
		user, err := service.CreateUser(newUser.Username, newUser.Email, newUser.Password)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Couldn't create user", "errors": err.Error()})
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
		token := c.Locals("user").(*jwt.Token)
		if !validToken(token, id) {
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Invalid token id", "data": nil})
		}

		service := service.NewUserService(uowFactory)
		user, err := service.GetUser(id)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		err = service.UpdateUser(user)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to update user", "data": nil})
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
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		token := c.Locals("user").(*jwt.Token)
		if !validToken(token, id) {
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Invalid token id", "data": nil})
		}
		service := service.NewUserService(uowFactory)
		if !service.ValidUser(id, pi.Password) {
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Not valid user", "data": nil})
		}

		err = service.DeleteUser(id, pi.Password)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to delete user", "data": nil})
		}
		return c.JSON(fiber.Map{"status": "success", "message": "User successfully deleted", "data": nil})
	}
}
