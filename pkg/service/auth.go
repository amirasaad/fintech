package service

import (
	"errors"
	"net/mail"
	"os"
	"time"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthStrategy interface {
	Login(identity, password string) (*domain.User, string, error)
}

type AuthService struct {
	uowFactory func() (repository.UnitOfWork, error)
	strategy   AuthStrategy
}

func NewAuthService(uowFactory func() (repository.UnitOfWork, error)) *AuthService {
	return &AuthService{uowFactory: uowFactory, strategy: &JWTAuthStrategy{uowFactory: uowFactory}}
}

func NewBasicAuthService(uowFactory func() (repository.UnitOfWork, error)) *AuthService {
	return &AuthService{uowFactory: uowFactory, strategy: &BasicAuthStrategy{uowFactory: uowFactory}}
}

func (s *AuthService) CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (s *AuthService) ValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func (s *AuthService) GetCurrentUserId(token *jwt.Token) (uuid.UUID, error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, domain.ErrUserUnauthorized
	}
	userIDRaw, ok := claims["user_id"].(string)
	if !ok {
		return uuid.Nil, domain.ErrUserUnauthorized
	}
	return uuid.Parse(userIDRaw)
}

func (s *AuthService) Login(identity, password string) (*domain.User, string, error) {
	return s.strategy.Login(identity, password)
}

// JWTAuthStrategy implements AuthStrategy for JWT-based authentication
type JWTAuthStrategy struct {
	uowFactory func() (repository.UnitOfWork, error)
}

func (s *JWTAuthStrategy) Login(identity, password string) (*domain.User, string, error) {
	uow, err := s.uowFactory()
	if err != nil {
		return nil, "", err
	}
	var user *domain.User
	if isEmail(identity) {
		user, err = uow.UserRepository().GetByEmail(identity)
	} else {
		user, err = uow.UserRepository().GetByUsername(identity)
	}
	const dummyHash = "$2a$10$7zFqzDbD3RrlkMTczbXG9OWZ0FLOXjIxXzSZ.QZxkVXjXcx7QZQiC"
	if err != nil {
		return nil, "", err
	}
	if user == nil {
		checkPasswordHash(password, dummyHash)
		return nil, "", nil
	}
	if !checkPasswordHash(password, user.Password) {
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

// BasicAuthStrategy implements AuthStrategy for CLI (no JWT, just password check)
type BasicAuthStrategy struct {
	uowFactory func() (repository.UnitOfWork, error)
}

func (s *BasicAuthStrategy) Login(identity, password string) (*domain.User, string, error) {
	uow, err := s.uowFactory()
	if err != nil {
		return nil, "", err
	}
	var user *domain.User
	if isEmail(identity) {
		user, err = uow.UserRepository().GetByEmail(identity)
	} else {
		user, err = uow.UserRepository().GetByUsername(identity)
	}
	const dummyHash = "$2a$10$7zFqzDbD3RrlkMTczbXG9OWZ0FLOXjIxXzSZ.QZxkVXjXcx7QZQiC"
	if err != nil {
		return nil, "", err
	}
	if user == nil {
		checkPasswordHash(password, dummyHash)
		return nil, "", nil
	}
	if !checkPasswordHash(password, user.Password) {
		return nil, "", nil
	}
	return user, "", nil // No JWT token for CLI
}

// Helper functions
func isEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func checkPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
