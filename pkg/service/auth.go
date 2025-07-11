package service

import (
	"context"
	"errors"
	"log/slog"
	"net/mail"
	"time"

	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/user"
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
	s.logger.Info("CheckPasswordHash called")
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		s.logger.Error("Password hash check failed", "error", err)
	}
	return err == nil
}

func (s *AuthService) ValidEmail(email string) bool {
	s.logger.Info("ValidEmail called", "email", email)
	_, err := mail.ParseAddress(email)
	if err != nil {
		s.logger.Error("Email validation failed", "email", email, "error", err)
	}
	return err == nil
}

func (s *AuthService) GetCurrentUserId(
	token *jwt.Token,
) (userID uuid.UUID, err error) {
	s.logger.Info("GetCurrentUserId called")
	userID, err = s.strategy.GetCurrentUserID(context.WithValue(context.TODO(), userContextKey, token))
	if err != nil {
		s.logger.Error("GetCurrentUserId failed", "error", err)
	}
	return
}

func (s *AuthService) Login(
	identity, password string,
) (u *domain.User, err error) {
	s.logger.Info("Login called", "identity", identity)
	u, err = s.strategy.Login(identity, password)
	if err != nil {
		s.logger.Error("Login failed", "identity", identity, "error", err)
		return
	}
	if u == nil {
		err = user.ErrUserUnauthorized
		s.logger.Error("Login failed", "identity", identity, "error", "user is nil")
		return
	}
	s.logger.Info("Login successful", "userID", u.ID)
	return
}

func (s *AuthService) GenerateToken(user *domain.User) (string, error) {
	s.logger.Info("GenerateToken called", "userID", user.ID)
	token, err := s.strategy.GenerateToken(user)
	if err != nil {
		s.logger.Error("GenerateToken failed", "userID", user.ID, "error", err)
		return "", err
	}
	s.logger.Info("GenerateToken successful", "userID", user.ID)
	return token, nil
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
	s.logger.Info("GenerateToken called", "userID", user.ID)
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["username"] = user.Username
	claims["email"] = user.Email
	claims["user_id"] = user.ID.String()
	claims["exp"] = time.Now().Add(s.cfg.Expiry).Unix()
	tokenString, err := token.SignedString([]byte(s.cfg.Secret))
	if err != nil {
		s.logger.Error("GenerateToken failed", "userID", user.ID, "error", err)
		return "", err
	}
	s.logger.Info("GenerateToken successful", "userID", user.ID)
	return tokenString, nil
}

func (s *JWTAuthStrategy) Login(
	identity, password string,
) (
	u *domain.User,
	err error,
) {
	s.logger.Info("Login called", "identity", identity)
	uow, err := s.uowFactory()
	if err != nil {
		s.logger.Error("Login failed", "identity", identity, "error", err)
		return
	}

	userRepo, err := uow.UserRepository()
	if err != nil {
		s.logger.Error("Login failed", "identity", identity, "error", err)
		return
	}

	if isEmail(identity) {
		u, err = userRepo.GetByEmail(identity)
	} else {
		u, err = userRepo.GetByUsername(identity)
	}
	const dummyHash = "$2a$10$7zFqzDbD3RrlkMTczbXG9OWZ0FLOXjIxXzSZ.QZxkVXjXcx7QZQiC"
	if err != nil {
		s.logger.Error("Login failed", "identity", identity, "error", err)
		return
	}
	if u == nil {
		err = user.ErrUserUnauthorized
		s.logger.Error("Login failed", "identity", identity, "error", domain.ErrUserUnauthorized)
		checkPasswordHash(password, dummyHash)
		return
	}
	if !checkPasswordHash(password, u.Password) {
		err = user.ErrUserUnauthorized
		s.logger.Error("Login failed", "identity", identity, "error", domain.ErrUserUnauthorized)
		return
	}
	s.logger.Info("Login successful", "userID", u.ID)
	return
}

func (s *JWTAuthStrategy) GetCurrentUserID(
	ctx context.Context,
) (userID uuid.UUID, err error) {
	token, ok := ctx.Value(userContextKey).(*jwt.Token)
	if !ok || token == nil {
		s.logger.Error("GetCurrentUserID failed", "error", domain.ErrUserUnauthorized)
		err = domain.ErrUserUnauthorized
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		s.logger.Error("GetCurrentUserID failed", "error", domain.ErrUserUnauthorized)
		err = domain.ErrUserUnauthorized
		return
	}
	userIDRaw, ok := claims["user_id"].(string)
	if !ok {
		s.logger.Error("GetCurrentUserID failed", "error", domain.ErrUserUnauthorized)
		err = domain.ErrUserUnauthorized
		return
	}
	userID, err = uuid.Parse(userIDRaw)
	if err != nil {
		s.logger.Error("GetCurrentUserID failed", "error", err)
	}
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
	s.logger.Info("Login called", "identity", identity)
	uow, err := s.uowFactory()
	if err != nil {
		s.logger.Error("Login failed", "identity", identity, "error", err)
		return
	}
	userRepo, err := uow.UserRepository()
	if err != nil {
		s.logger.Error("Login failed", "identity", identity, "error", err)
		return
	}
	if isEmail(identity) {
		user, err = userRepo.GetByEmail(identity)
	} else {
		user, err = userRepo.GetByUsername(identity)
	}
	const dummyHash = "$2a$10$7zFqzDbD3RrlkMTczbXG9OWZ0FLOXjIxXzSZ.QZxkVXjXcx7QZQiC"
	if err != nil {
		s.logger.Error("Login failed", "identity", identity, "error", err)
		return
	}
	if user == nil {
		s.logger.Error("Login failed", "identity", identity, "error", errors.New("invalid credentials"))
		checkPasswordHash(password, dummyHash)
		return
	}
	if !checkPasswordHash(password, user.Password) {
		s.logger.Error("Login failed", "identity", identity, "error", errors.New("invalid credentials"))
		return
	}
	s.logger.Info("Login successful", "userID", user.ID)
	return
}

func (s *BasicAuthStrategy) GetCurrentUserID(ctx context.Context) (uuid.UUID, error) {
	s.logger.Info("GetCurrentUserID called")
	return uuid.Nil, nil
}

func (s *BasicAuthStrategy) GenerateToken(user *domain.User) (string, error) {
	s.logger.Info("GenerateToken called", "userID", user.ID)
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
