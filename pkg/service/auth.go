package service

import (
	"context"
	"net/mail"
	"time"

	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const userContextKey contextKey = "user"

type AuthStrategy interface {
	Login(identity, password string) (*domain.User, string, error)
	GetCurrentUserID(ctx context.Context) (uuid.UUID, error)
}

type AuthService struct {
	uowFactory func() (repository.UnitOfWork, error)
	strategy   AuthStrategy
}

func NewAuthService(
	uowFactory func() (repository.UnitOfWork, error),
	strategy AuthStrategy,
) *AuthService {
	return &AuthService{uowFactory: uowFactory, strategy: strategy}
}

func NewBasicAuthService(uowFactory func() (repository.UnitOfWork, error)) *AuthService {
	return NewAuthService(uowFactory, &BasicAuthStrategy{uowFactory: uowFactory})
}

func (s *AuthService) CheckPasswordHash(
	password, hash string,
) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (s *AuthService) ValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func (s *AuthService) GetCurrentUserId(
	token *jwt.Token,
) (userID uuid.UUID, err error) {
	userID, err = s.strategy.GetCurrentUserID(context.WithValue(context.TODO(), userContextKey, token))
	return
}

func (s *AuthService) Login(
	identity, password string,
) (user *domain.User, token string, err error) {
	user, token, err = s.strategy.Login(identity, password)
	return
}

// JWTAuthStrategy implements AuthStrategy for JWT-based authentication
type JWTAuthStrategy struct {
	uowFactory func() (repository.UnitOfWork, error)
	cfg        config.AuthConfig
}

func NewJWTAuthStrategy(
	uowFactory func() (repository.UnitOfWork, error),
	cfg config.AuthConfig,
) *JWTAuthStrategy {
	return &JWTAuthStrategy{uowFactory: uowFactory, cfg: cfg}
}

func (s *JWTAuthStrategy) Login(
	identity, password string,
) (
	user *domain.User,
	tokenString string,
	err error,
) {
	uow, err := s.uowFactory()
	if err != nil {
		return
	}
	if isEmail(identity) {
		user, err = uow.UserRepository().GetByEmail(identity)
	} else {
		user, err = uow.UserRepository().GetByUsername(identity)
	}
	const dummyHash = "$2a$10$7zFqzDbD3RrlkMTczbXG9OWZ0FLOXjIxXzSZ.QZxkVXjXcx7QZQiC"
	if err != nil {
		return
	}
	if user == nil {
		checkPasswordHash(password, dummyHash)
		return
	}
	if !checkPasswordHash(password, user.Password) {
		return
	}

	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["username"] = user.Username
	claims["email"] = user.Email
	claims["user_id"] = user.ID.String()
	claims["exp"] = time.Now().Add(s.cfg.JwtExpiry).Unix()
	tokenString, err = token.SignedString([]byte(s.cfg.JwtSecret))
	if err != nil {
		return
	}
	return
}

func (s *JWTAuthStrategy) GetCurrentUserID(
	ctx context.Context,
) (userID uuid.UUID, err error) {
	token := ctx.Value(userContextKey).(*jwt.Token)
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		err = domain.ErrUserUnauthorized
		return
	}
	userIDRaw, ok := claims["user_id"].(string)
	if !ok {
		err = domain.ErrUserUnauthorized
		return
	}
	userID, err = uuid.Parse(userIDRaw)
	return
}

// BasicAuthStrategy implements AuthStrategy for CLI (no JWT, just password check)
type BasicAuthStrategy struct {
	uowFactory func() (repository.UnitOfWork, error)
}

func (s *BasicAuthStrategy) Login(
	identity, password string,
) (
	user *domain.User,
	tokenString string,
	err error,
) {
	uow, err := s.uowFactory()
	if err != nil {
		return
	}
	if isEmail(identity) {
		user, err = uow.UserRepository().GetByEmail(identity)
	} else {
		user, err = uow.UserRepository().GetByUsername(identity)
	}
	const dummyHash = "$2a$10$7zFqzDbD3RrlkMTczbXG9OWZ0FLOXjIxXzSZ.QZxkVXjXcx7QZQiC"
	if err != nil {
		return
	}
	if user == nil {
		checkPasswordHash(password, dummyHash)
		return
	}
	if !checkPasswordHash(password, user.Password) {
		return
	}
	return
}

func (s *BasicAuthStrategy) GetCurrentUserID(ctx context.Context) (uuid.UUID, error) {
	return uuid.Nil, nil
}

// Helper functions
func isEmail(
	email string,
) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func checkPasswordHash(
	password, hash string,
) (isValid bool) {
	isValid = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
	return
}
