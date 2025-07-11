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

// AuthService provides authentication and user validation logic.
type AuthService struct {
	uow    repository.UnitOfWork
	logger *slog.Logger
}

// NewAuthService creates a new AuthService with a UnitOfWork and logger.
func NewAuthService(uow repository.UnitOfWork, logger *slog.Logger) *AuthService {
	return &AuthService{uow: uow, logger: logger}
}

// NewBasicAuthService creates a new AuthService with a UnitOfWork and logger.
func NewBasicAuthService(uow repository.UnitOfWork, logger *slog.Logger) *AuthService {
	return NewAuthService(uow, logger)
}

// CheckPasswordHash verifies a password against a hash.
func (s *AuthService) CheckPasswordHash(password, hash string) bool {
	s.logger.Info("CheckPasswordHash called")
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		s.logger.Error("Password hash check failed", "error", err)
	}
	return err == nil
}

// ValidEmail checks if the email is valid.
func (s *AuthService) ValidEmail(email string) bool {
	s.logger.Info("ValidEmail called", "email", email)
	_, err := mail.ParseAddress(email)
	if err != nil {
		s.logger.Error("Email validation failed", "email", email, "error", err)
	}
	return err == nil
}

// Login authenticates a user by identity and password in a transaction.
func (s *AuthService) Login(ctx context.Context, identity, password string) (u *domain.User, err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repo, err := uow.GetRepository[repository.UserRepository]()
		if err != nil {
			return err
		}
		if isEmail(identity) {
			u, err = repo.GetByEmail(identity)
		} else {
			u, err = repo.GetByUsername(identity)
		}
		if err != nil {
			return err
		}
		if u == nil {
			return user.ErrUserUnauthorized
		}
		if !s.CheckPasswordHash(password, u.Password) {
			return user.ErrUserUnauthorized
		}
		return nil
	})
	return
}

// GenerateToken generates a JWT token for a user.
func (s *AuthService) GenerateToken(user *domain.User, cfg config.JwtConfig) (string, error) {
	s.logger.Info("GenerateToken called", "userID", user.ID)
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["username"] = user.Username
	claims["email"] = user.Email
	claims["user_id"] = user.ID.String()
	claims["exp"] = time.Now().Add(cfg.Expiry).Unix()
	tokenString, err := token.SignedString([]byte(cfg.Secret))
	if err != nil {
		s.logger.Error("GenerateToken failed", "userID", user.ID, "error", err)
		return "", err
	}
	s.logger.Info("GenerateToken successful", "userID", user.ID)
	return tokenString, nil
}

// Helper function
func isEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
