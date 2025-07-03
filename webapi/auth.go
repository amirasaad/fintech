package webapi

import (
	"net/mail"
	"time"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func AuthRoutes(app *fiber.App, uowFactory func() (repository.UnitOfWork, error)) {
	app.Post("/login", Login(uowFactory))
}

// CheckPasswordHash compare password with hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	return err == nil
}

func valid(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// Login get user and password
func Login(uowFactory func() (repository.UnitOfWork, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		type LoginInput struct {
			Identity string `json:"identity"`
			Password string `json:"password"`
		}
		input := new(LoginInput)

		if err := c.BodyParser(input); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Error on login request", "errors": err.Error()})
		}

		identity := input.Identity
		pass := input.Password
		uow, err := uowFactory()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "	error", "message": "Failed to create unit of work", "data": nil})
		}
		defer uow.Rollback()
		var user *domain.User
		if valid(identity) {
			user, err = uow.UserRepository().GetByEmail(identity)
		} else {
			user, err = uow.UserRepository().GetByUsername(identity)
		}

		const dummyHash = "$2a$10$7zFqzDbD3RrlkMTczbXG9OWZ0FLOXjIxXzSZ.QZxkVXjXcx7QZQiC" // => Hashed " "

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Internal Server Error", "data": err})
		}
		if user == nil {
			// Always perform a hash check, even if the user doesn't exist, to prevent timing attacks
			CheckPasswordHash(pass, dummyHash)

			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Invalid identity or password", "data": err})
		}

		if !CheckPasswordHash(pass, user.Password) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Invalid identity or password", "data": nil})
		}

		token := jwt.New(jwt.SigningMethodHS256)

		claims := token.Claims.(jwt.MapClaims)
		claims["username"] = user.Username
		claims["email"] = user.Email
		claims["user_id"] = user.ID
		claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

		t, err := token.SignedString([]byte("SECRET"))

		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.JSON(fiber.Map{"status": "success", "message": "Success login", "token": t})
	}
}
