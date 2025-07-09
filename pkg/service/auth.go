package service

import (
	"context"
	"errors"
	"log/slog"
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
	Login(identity, password string) (*domain.User, error)
	GetCurrentUserID(ctx context.Context) (uuid.UUID, error)
	GenerateToken(user *domain.User) (string, error)
}

type AuthService struct {
	uowFactory func() (repository.UnitOfWork, error)
	strategy   AuthStrategy
	logger     *slog.Logger
}

func NewAuthService(
	uowFactory func() (repository.UnitOfWork, error),
	strategy AuthStrategy,
	logger *slog.Logger,
) *AuthService {
	return &AuthService{uowFactory: uowFactory, strategy: strategy, logger: logger}
}

func NewBasicAuthService(uowFactory func() (repository.UnitOfWork, error), logger *slog.Logger) *AuthService {
	return NewAuthService(uowFactory, &BasicAuthStrategy{uowFactory: uowFactory}, logger)
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
) (user *domain.User, err error) {
	user, err = s.strategy.Login(identity, password)
	if err != nil {
		return
	}
	return
}

func (s *AuthService) GenerateToken(user *domain.User) (string, error) {
	return s.strategy.GenerateToken(user)
}

// JWTAuthStrategy implements AuthStrategy for JWT-based authentication
type JWTAuthStrategy struct {
	uowFactory func() (repository.UnitOfWork, error)
	cfg        config.JwtConfig
	logger     *slog.Logger
}

func NewJWTAuthStrategy(
	uowFactory func() (repository.UnitOfWork, error),
	cfg config.JwtConfig,
	logger *slog.Logger,
) *JWTAuthStrategy {
	return &JWTAuthStrategy{uowFactory: uowFactory, cfg: cfg, logger: logger}
}
func (s *JWTAuthStrategy) GenerateToken(user *domain.User) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["username"] = user.Username
	claims["email"] = user.Email
	claims["user_id"] = user.ID.String()
	claims["exp"] = time.Now().Add(s.cfg.Expiry).Unix()
	return token.SignedString([]byte(s.cfg.Secret))
}

func (s *JWTAuthStrategy) Login(
	identity, password string,
) (
	user *domain.User,
	err error,
) {
	uow, err := s.uowFactory()
	if err != nil {
		return
	}

	userRepo, err := uow.UserRepository()
	if err != nil {
		return
	}

	if isEmail(identity) {
		user, err = userRepo.GetByEmail(identity)
	} else {
		user, err = userRepo.GetByUsername(identity)
	}
	const dummyHash = "$2a$10$7zFqzDbD3RrlkMTczbXG9OWZ0FLOXjIxXzSZ.QZxkVXjXcx7QZQiC"
	if err != nil {
		return
	}
	if user == nil {
		err = domain.ErrUserUnauthorized
		checkPasswordHash(password, dummyHash)
		return
	}
	if !checkPasswordHash(password, user.Password) {
		err = domain.ErrUserUnauthorized
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
	logger     *slog.Logger
}

func (s *BasicAuthStrategy) Login(
	identity, password string,
) (
	user *domain.User,
	err error,
) {
	uow, err := s.uowFactory()
	if err != nil {
		return
	}
	userRepo, err := uow.UserRepository()
	if err != nil {
		return
	}
	if isEmail(identity) {
		user, err = userRepo.GetByEmail(identity)
	} else {
		user, err = userRepo.GetByUsername(identity)
	}
	const dummyHash = "$2a$10$7zFqzDbD3RrlkMTczbXG9OWZ0FLOXjIxXzSZ.QZxkVXjXcx7QZQiC"
	if err != nil {
		return
	}
	if user == nil {
		checkPasswordHash(password, dummyHash)
		err = errors.New("invalid credentials")
		return
	}
	if !checkPasswordHash(password, user.Password) {
		err = errors.New("invalid credentials")
		return
	}
	return
}

func (s *BasicAuthStrategy) GetCurrentUserID(ctx context.Context) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (s *BasicAuthStrategy) GenerateToken(user *domain.User) (string, error) {
	return "", nil // No token for basic auth
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
