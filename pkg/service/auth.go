package service

import (
	"errors"
	"net/mail"
	"os"
	"time"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	uowFactory func() (repository.UnitOfWork, error)
}

func NewAuthService(uowFactory func() (repository.UnitOfWork, error)) *AuthService {
	return &AuthService{uowFactory: uowFactory}
}

func (s *AuthService) CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (s *AuthService) ValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func (s *AuthService) Login(identity, password string) (*domain.User, string, error) {
	uow, err := s.uowFactory()
	if err != nil {
		return nil, "", err
	}
	var user *domain.User
	if s.ValidEmail(identity) {
		user, err = uow.UserRepository().GetByEmail(identity)
	} else {
		user, err = uow.UserRepository().GetByUsername(identity)
	}
	const dummyHash = "$2a$10$7zFqzDbD3RrlkMTczbXG9OWZ0FLOXjIxXzSZ.QZxkVXjXcx7QZQiC" // Hashed " "
	if err != nil {
		return nil, "", err
	}
	if user == nil {
		s.CheckPasswordHash(password, dummyHash)
		return nil, "", nil
	}
	if !s.CheckPasswordHash(password, user.Password) {
		return nil, "", nil
	}
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		return nil, "", errors.New("JWT secret key is not set")
	}
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["username"] = user.Username
	claims["email"] = user.Email
	claims["user_id"] = user.ID.String()
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return nil, "", err
	}
	return user, tokenString, nil
}
