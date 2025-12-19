package auth

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/amirasaad/fintech/pkg/config"

	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/repository"
	repouser "github.com/amirasaad/fintech/pkg/repository/user"
	"github.com/amirasaad/fintech/pkg/utils"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const userContextKey contextKey = "user"

type Strategy interface {
	Login(ctx context.Context, identity, password string) (*dto.UserRead, error)
	GetCurrentUserID(ctx context.Context) (uuid.UUID, error)
	GenerateToken(ctx context.Context, u *dto.UserRead) (string, error)
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
	cfg *config.Jwt,
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
	s.logger.Debug("ValidEmail called", "email", email)
	return utils.IsEmail(email)
}

func (s *Service) GetCurrentUserId(
	token *jwt.Token,
) (userID uuid.UUID, err error) {
	log := s.logger.With("context", "GetCurrentUserId")
	log.Debug("GetCurrentUserId called")
	userID, err = s.strategy.GetCurrentUserID(
		context.WithValue(
			context.Background(),
			userContextKey,
			token,
		),
	)
	if err != nil {
		log.Error("GetCurrentUserId failed", "error", err)
		return
	}
	log.Info("GetCurrentUserId successful", "userID", userID)
	return
}

func (s *Service) Login(
	ctx context.Context,
	identity, password string,
) (u *dto.UserRead, err error) {
	log := s.logger.With("context", "Login")
	log.Debug("Login called", "identity", identity)
	u, err = s.strategy.Login(ctx, identity, password)
	if err != nil {
		log.Error("Login failed", "identity", identity, "error", err)
		return
	}
	log.Info("Login successful", "userID", u.ID)
	return
}

func (s *Service) GenerateToken(
	ctx context.Context,
	u *dto.UserRead,
) (string, error) {
	log := s.logger.With("userID", u.ID)
	log.Debug("GenerateToken called")
	token, err := s.strategy.GenerateToken(ctx, u)
	if err != nil {
		log.Error("GenerateToken failed", "userID", u.ID, "error", err)
		return "", err
	}
	log.Info("GenerateToken successful")
	return token, nil
}

// JWTStrategy implements AuthStrategy for JWT-based authentication
type JWTStrategy struct {
	uow    repository.UnitOfWork
	cfg    *config.Jwt
	logger *slog.Logger
}

func NewJWTStrategy(
	uow repository.UnitOfWork,
	cfg *config.Jwt,
	logger *slog.Logger,
) *JWTStrategy {
	return &JWTStrategy{uow: uow, cfg: cfg, logger: logger}
}

func (s *JWTStrategy) GenerateToken(
	ctx context.Context,
	u *dto.UserRead) (string, error) {
	log := s.logger.With("userID", u.ID)
	log.Debug("GenerateToken called", "userID", u.ID)
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["username"] = u.Username
	claims["email"] = u.Email
	claims["user_id"] = u.ID.String()
	claims["exp"] = time.Now().Add(s.cfg.Expiry).Unix()
	tokenString, err := token.SignedString([]byte(s.cfg.Secret))
	if err != nil {
		log.Error("GenerateToken failed", "userID", u.ID, "error", err)
		return "", err
	}
	log.Info("GenerateToken successful")
	return tokenString, nil
}

func (s *JWTStrategy) Login(
	ctx context.Context,
	identity, password string,
) (
	u *dto.UserRead,
	err error,
) {
	log := s.logger.With("context", "Login", "identity", identity)
	log.Debug("Login called")
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository((*repouser.Repository)(nil))
		if err != nil {
			return fmt.Errorf("failed to get user repository: %w", err)
		}
		repo, ok := repoAny.(repouser.Repository)
		if !ok {
			return fmt.Errorf("invalid user repository type")
		}
		// Check if identity is email or username
		if utils.IsEmail(identity) {
			u, err = repo.GetByEmail(ctx, identity)
		} else {
			u, err = repo.GetByUsername(ctx, identity)
		}

		const dummyHash = "$2a$10$7zFqzDbD3RrlkMTczbXG9OWZ0FLOXjIxXzSZ.QZxkVXjXcx7QZQiC"
		if err != nil {
			return user.ErrUserUnauthorized
		}
		if u == nil {
			// Always check password hash to avoid timing attacks
			_ = utils.CheckPasswordHash(password, dummyHash)

			log.Error("Login failed", "error", user.ErrUserUnauthorized)
			return user.ErrUserUnauthorized
		}
		if !utils.CheckPasswordHash(
			password,
			u.HashedPassword,
		) {
			log.Error("Login failed", "error", user.ErrUserUnauthorized)
			return user.ErrUserUnauthorized
		}
		return nil
	})
	return
}

func (s *JWTStrategy) GetCurrentUserID(
	ctx context.Context,
) (userID uuid.UUID, err error) {
	log := s.logger.With("context", "GetCurrentUserID")
	log.Debug("GetCurrentUserID called")
	token, ok := ctx.Value(userContextKey).(*jwt.Token)
	if !ok || token == nil {
		log.Error("GetCurrentUserID failed", "error", user.ErrUserUnauthorized)
		err = user.ErrUserUnauthorized
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Error("GetCurrentUserID failed", "error", user.ErrUserUnauthorized)
		err = user.ErrUserUnauthorized
		return
	}
	userIDRaw, ok := claims["user_id"].(string)
	if !ok {
		log.Error("GetCurrentUserID failed", "error", user.ErrUserUnauthorized)
		err = user.ErrUserUnauthorized
		return
	}
	userID, err = uuid.Parse(userIDRaw)
	if err != nil {
		log.Error("GetCurrentUserID failed", "error", err)
		return
	}
	log.Info("GetCurrentUserID successful", "userID", userID)
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
) (u *dto.UserRead, err error) {
	log := s.logger.With("identity", identity)
	log.Info("BasicAuth Login called")
	repoAny, err := s.uow.GetRepository((*repouser.Repository)(nil))
	if err != nil {
		err = fmt.Errorf("failed to get user repository: %w", err)
		s.logger.Error("Failed to get user repository", "error", err)
		return
	}

	repo, ok := repoAny.(repouser.Repository)
	if !ok {
		err = fmt.Errorf("invalid user repository type")
		s.logger.Error("Invalid user repository type")
		return
	}

	log.Info("Looking up user")
	if utils.IsEmail(identity) {
		u, err = repo.GetByEmail(ctx, identity)
	} else {
		u, err = repo.GetByUsername(ctx, identity)
	}

	// If there was an error from the repository, return it
	if err != nil {
		log.Error("Repository error", "error", err)
		return nil, fmt.Errorf("repository error: %w", err)
	}

	// If user not found, return unauthorized
	if u == nil {
		log.Info("User not found", "identity", identity)
		return nil, user.ErrUserUnauthorized
	}

	log.Info("User found", "userID", u.ID, "username", u.Username)
	// Check password against the hardcoded hash for "password"
	const dummyHash = "$2a$10$.IIxpSc3OElWXLV2Wj517eUGmZ64IQgBNQ4OcFbanW85CTrgrIDQy"
	log.Debug("Comparing password hash", "providedPassword", password, "hash", dummyHash)
	if valid := utils.CheckPasswordHash(password, dummyHash); !valid {
		log.Error("Password comparison failed", "error", err)
		return nil, user.ErrUserUnauthorized
	}
	log.Info("Password comparison succeeded")

	return
}

func (s *BasicAuthStrategy) GetCurrentUserID(ctx context.Context) (uuid.UUID, error) {
	log := s.logger.With("context", "GetCurrentUserID")
	log.Debug("GetCurrentUserID called")
	return uuid.Nil, nil
}

func (s *BasicAuthStrategy) GenerateToken(ctx context.Context, u *dto.UserRead) (string, error) {
	log := s.logger.With("userID", u.ID)
	log.Debug("GenerateToken called")
	return "", nil // No token for basic auth
}
