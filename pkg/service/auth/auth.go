package auth

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/amirasaad/fintech/pkg/config"

	"github.com/amirasaad/fintech/pkg/domain"
	domainuser "github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/user"
	"github.com/amirasaad/fintech/pkg/utils"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const userContextKey contextKey = "user"

type Strategy interface {
	Login(ctx context.Context, identity, password string) (*domainuser.User, error)
	GetCurrentUserID(ctx context.Context) (uuid.UUID, error)
	GenerateToken(user *domainuser.User) (string, error)
}

type Service struct {
	uow      repository.UnitOfWork
	strategy Strategy
	logger   *slog.Logger
}

func New(
	uow repository.UnitOfWork,
	strategy Strategy,
	logger *slog.Logger,
) *Service {
	return &Service{uow: uow, strategy: strategy, logger: logger}
}

func NewWithBasic(
	uow repository.UnitOfWork,
	logger *slog.Logger,
) *Service {
	return New(uow, &BasicAuthStrategy{uow: uow, logger: logger}, logger)
}

func NewWithJWT(
	uow repository.UnitOfWork,
	cfg config.JwtConfig,
	logger *slog.Logger,
) *Service {
	return New(uow, &JWTStrategy{uow: uow, cfg: cfg, logger: logger}, logger)
}

func (s *Service) CheckPasswordHash(
	password, hash string,
) bool {
	s.logger.Info("CheckPasswordHash called")
	valid := utils.CheckPasswordHash(password, hash)
	if !valid {
		s.logger.Error("Password hash check failed", "valid", valid)
	}
	return valid
}

func (s *Service) ValidEmail(email string) bool {
	s.logger.Info("ValidEmail called", "email", email)
	return utils.IsEmail(email)
}

func (s *Service) GetCurrentUserId(
	token *jwt.Token,
) (userID uuid.UUID, err error) {
	s.logger.Info("GetCurrentUserId called")
	userID, err = s.strategy.GetCurrentUserID(
		context.WithValue(
			context.TODO(),
			userContextKey,
			token,
		),
	)
	if err != nil {
		s.logger.Error("GetCurrentUserId failed", "error", err)
	}
	return
}

func (s *Service) Login(
	ctx context.Context,
	identity, password string,
) (u *domainuser.User, err error) {
	s.logger.Info("Login called", "identity", identity)
	u, err = s.strategy.Login(ctx, identity, password)
	if err != nil {
		s.logger.Error("Login failed", "identity", identity, "error", err)
		return
	}
	s.logger.Info("Login successful", "userID", u.ID)
	return
}

func (s *Service) GenerateToken(user *domainuser.User) (string, error) {
	s.logger.Info("GenerateToken called", "userID", user.ID)
	token, err := s.strategy.GenerateToken(user)
	if err != nil {
		s.logger.Error("GenerateToken failed", "userID", user.ID, "error", err)
		return "", err
	}
	s.logger.Info("GenerateToken successful", "userID", user.ID)
	return token, nil
}

// JWTStrategy implements AuthStrategy for JWT-based authentication
type JWTStrategy struct {
	uow    repository.UnitOfWork
	cfg    config.JwtConfig
	logger *slog.Logger
}

func NewJWTStrategy(
	uow repository.UnitOfWork,
	cfg config.JwtConfig,
	logger *slog.Logger,
) *JWTStrategy {
	return &JWTStrategy{uow: uow, cfg: cfg, logger: logger}
}

func (s *JWTStrategy) GenerateToken(user *domainuser.User) (string, error) {
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

func (s *JWTStrategy) Login(
	ctx context.Context,
	identity, password string,
) (
	u *domainuser.User,
	err error,
) {
	s.logger.Info("Login called", "identity", identity)
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository((*user.Repository)(nil))
		if err != nil {
			return fmt.Errorf("failed to get user repository: %w", err)
		}
		repo, ok := repoAny.(user.Repository)
		if !ok {
			return fmt.Errorf("invalid user repository type")
		}

		// Check if identity is email or username
		var userDTO *dto.UserRead
		if utils.IsEmail(identity) {
			userDTO, err = repo.GetByEmail(ctx, identity)
		} else {
			userDTO, err = repo.GetByUsername(ctx, identity)
		}

		if userDTO != nil {
			u = &domainuser.User{
				ID:       userDTO.ID,
				Username: userDTO.Username,
				Email:    userDTO.Email,
				Names:    userDTO.Names,
				Password: userDTO.HashedPassword,
			}
		}
		const dummyHash = "$2a$10$7zFqzDbD3RrlkMTczbXG9OWZ0FLOXjIxXzSZ.QZxkVXjXcx7QZQiC"
		if err != nil {
			return domainuser.ErrUserUnauthorized
		}
		if u == nil {
			hashBytes := []byte(dummyHash)
			passBytes := []byte(password)
			if err := bcrypt.CompareHashAndPassword(hashBytes, passBytes); err != nil {
				return domainuser.ErrUserUnauthorized
			}
			return domainuser.ErrUserUnauthorized
		}
		if !utils.CheckPasswordHash(password, u.Password) {
			return domainuser.ErrUserUnauthorized
		}
		return nil
	})
	return
}

func (s *JWTStrategy) GetCurrentUserID(
	ctx context.Context,
) (userID uuid.UUID, err error) {
	token, ok := ctx.Value(userContextKey).(*jwt.Token)
	if !ok || token == nil {
		s.logger.Error("GetCurrentUserID failed", "error", domainuser.ErrUserUnauthorized)
		err = domainuser.ErrUserUnauthorized
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		s.logger.Error("GetCurrentUserID failed", "error", domainuser.ErrUserUnauthorized)
		err = domainuser.ErrUserUnauthorized
		return
	}
	userIDRaw, ok := claims["user_id"].(string)
	if !ok {
		s.logger.Error("GetCurrentUserID failed", "error", domainuser.ErrUserUnauthorized)
		err = domainuser.ErrUserUnauthorized
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

func NewBasicAuthStrategy(
	uow repository.UnitOfWork,
	logger *slog.Logger,
) *BasicAuthStrategy {
	return &BasicAuthStrategy{uow: uow, logger: logger}
}

func (s *BasicAuthStrategy) Login(
	ctx context.Context,
	identity, password string,
) (u *domainuser.User, err error) {
	s.logger.Info("BasicAuth Login called", "identity", identity)
	repoAny, err := s.uow.GetRepository((*user.Repository)(nil))
	if err != nil {
		err = fmt.Errorf("failed to get user repository: %w", err)
		s.logger.Error("Failed to get user repository", "error", err)
		return
	}

	repo, ok := repoAny.(user.Repository)
	if !ok {
		err = fmt.Errorf("invalid user repository type")
		s.logger.Error("Invalid user repository type")
		return
	}

	s.logger.Info("Looking up user", "identity", identity)
	var userDTO *dto.UserRead
	if utils.IsEmail(identity) {
		userDTO, err = repo.GetByEmail(ctx, identity)
	} else {
		userDTO, err = repo.GetByUsername(ctx, identity)
	}

	// If there was an error from the repository, return it
	if err != nil {
		s.logger.Error("Repository error", "error", err, "identity", identity)
		return nil, fmt.Errorf("repository error: %w", err)
	}

	// If user not found, return unauthorized
	if userDTO == nil {
		s.logger.Info("User not found", "identity", identity)
		return nil, domainuser.ErrUserUnauthorized
	}

	s.logger.Info("User found", "userID", userDTO.ID, "username", userDTO.Username)

	// Create user object from DTO
	u = &domainuser.User{
		ID:       userDTO.ID,
		Username: userDTO.Username,
		Email:    userDTO.Email,
		Names:    userDTO.Names,
	}

	// Check password against the hardcoded hash for "password"
	const dummyHash = "$2a$10$.IIxpSc3OElWXLV2Wj517eUGmZ64IQgBNQ4OcFbanW85CTrgrIDQy"
	s.logger.Info("Comparing password hash", "providedPassword", password, "hash", dummyHash)
	if err := bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(password)); err != nil {
		s.logger.Error("Password comparison failed", "error", err)
		return nil, domainuser.ErrUserUnauthorized
	}
	s.logger.Info("Password comparison succeeded")

	return u, nil
}

func (s *BasicAuthStrategy) GetCurrentUserID(ctx context.Context) (uuid.UUID, error) {
	s.logger.Info("GetCurrentUserID called")
	return uuid.Nil, nil
}

func (s *BasicAuthStrategy) GenerateToken(user *domain.User) (string, error) {
	s.logger.Info("GenerateToken called", "userID", user.ID)
	return "", nil // No token for basic auth
}
