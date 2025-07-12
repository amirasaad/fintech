package auth

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"time"

	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/utils"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const userContextKey contextKey = "user"

type AuthStrategy interface {
	Login(ctx context.Context, identity, password string) (*domain.User, error)
	GetCurrentUserID(ctx context.Context) (uuid.UUID, error)
	GenerateToken(user *domain.User) (string, error)
}

type AuthService struct {
	uow      repository.UnitOfWork
	strategy AuthStrategy
	logger   *slog.Logger
}

func NewAuthService(
	uow repository.UnitOfWork,
	strategy AuthStrategy,
	logger *slog.Logger,
) *AuthService {
	return &AuthService{uow: uow, strategy: strategy, logger: logger}
}

func NewBasicAuthService(uow repository.UnitOfWork, logger *slog.Logger) *AuthService {
	return NewAuthService(uow, &BasicAuthStrategy{uow: uow, logger: logger}, logger)
}

func (s *AuthService) CheckPasswordHash(
	password, hash string,
) bool {
	s.logger.Info("CheckPasswordHash called")
	valid := utils.CheckPasswordHash(password, hash)
	if !valid {
		s.logger.Error("Password hash check failed", "valid", valid)
	}
	return valid
}

func (s *AuthService) ValidEmail(email string) bool {
	s.logger.Info("ValidEmail called", "email", email)
	return utils.IsEmail(email)
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
	ctx context.Context,
	identity, password string,
) (u *domain.User, err error) {
	s.logger.Info("Login called", "identity", identity)
	u, err = s.strategy.Login(ctx, identity, password)
	if err != nil {
		s.logger.Error("Login failed", "identity", identity, "error", err)
		return
	}
	if u == nil {
		err = domain.ErrUserUnauthorized
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
	uow    repository.UnitOfWork
	cfg    config.JwtConfig
	logger *slog.Logger
}

func NewJWTAuthStrategy(
	uow repository.UnitOfWork,
	cfg config.JwtConfig,
	logger *slog.Logger,
) *JWTAuthStrategy {
	return &JWTAuthStrategy{uow: uow, cfg: cfg, logger: logger}
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
	ctx context.Context,
	identity, password string,
) (
	u *domain.User,
	err error,
) {
	s.logger.Info("Login called", "identity", identity)
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.UserRepository)(nil)).Elem())
		if err != nil {
			return err
		}
		repo := repoAny.(repository.UserRepository)
		if utils.IsEmail(identity) {
			u, err = repo.GetByEmail(identity)
		} else {
			u, err = repo.GetByUsername(identity)
		}
		const dummyHash = "$2a$10$7zFqzDbD3RrlkMTczbXG9OWZ0FLOXjIxXzSZ.QZxkVXjXcx7QZQiC"
		if err != nil {
			return domain.ErrUserUnauthorized
		}
		if u == nil {
			utils.CheckPasswordHash(password, dummyHash)
			return domain.ErrUserUnauthorized
		}
		if !utils.CheckPasswordHash(password, u.Password) {
			return domain.ErrUserUnauthorized
		}
		return nil
	})
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
	uow    repository.UnitOfWork
	logger *slog.Logger
}

func (s *BasicAuthStrategy) Login(
	ctx context.Context,
	identity, password string,
) (
	user *domain.User,
	err error,
) {
	s.logger.Info("Login called", "identity", identity)
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.UserRepository)(nil)).Elem())
		if err != nil {
			return err
		}
		repo := repoAny.(repository.UserRepository)
		if utils.IsEmail(identity) {
			user, err = repo.GetByEmail(identity)
		} else {
			user, err = repo.GetByUsername(identity)
		}
		const dummyHash = "$2a$10$7zFqzDbD3RrlkMTczbXG9OWZ0FLOXjIxXzSZ.QZxkVXjXcx7QZQiC"
		if err != nil {
			return domain.ErrUserUnauthorized
		}
		if user == nil {
			utils.CheckPasswordHash(password, dummyHash)
			return errors.New("invalid credentials")
		}
		if !utils.CheckPasswordHash(password, user.Password) {
			return errors.New("invalid credentials")
		}
		return nil
	})
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
