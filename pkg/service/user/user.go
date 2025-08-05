// Package user provides business logic for user management operations.
// It uses the decorator pattern for transaction management and includes comprehensive logging.
package user

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/repository"
	userrepo "github.com/amirasaad/fintech/pkg/repository/user"
	"github.com/amirasaad/fintech/pkg/utils"
	"github.com/google/uuid"
)

// Service provides business logic for user operations including creation, updates, and deletion.
type Service struct {
	uow    repository.UnitOfWork
	logger *slog.Logger
}

// New creates a new Service with a UnitOfWork and logger.
func New(
	uow repository.UnitOfWork,
	logger *slog.Logger,
) *Service {
	return &Service{
		uow:    uow,
		logger: logger,
	}
}

// CreateUser creates a new user account in a transaction.
func (s *Service) CreateUser(
	ctx context.Context,
	username, email, password string,
) (u *user.User, err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository((*userrepo.Repository)(nil))
		if err != nil {
			return err
		}
		repo, ok := repoAny.(userrepo.Repository)
		if !ok {
			return fmt.Errorf("unexpected repository type")
		}
		u, err = user.New(username, email, password)
		if err != nil {
			return err
		}
		return repo.Create(ctx, &dto.UserCreate{
			ID:       u.ID,
			Username: u.Username,
			Email:    u.Email,
			Password: u.Password,
		})
	})
	if err != nil {
		u = nil
	}
	return
}

// GetUser retrieves a user by ID in a transaction.
func (s *Service) GetUser(
	ctx context.Context,
	userID string,
) (u *dto.UserRead, err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository((*userrepo.Repository)(nil))
		if err != nil {
			return err
		}
		repo, ok := repoAny.(userrepo.Repository)
		if !ok {
			return fmt.Errorf("unexpected repository type")
		}
		uid, parseErr := uuid.Parse(userID)
		if parseErr != nil {
			return parseErr
		}
		u, err = repo.Get(ctx, uid)
		return err
	})
	if err != nil {
		u = nil
	}
	return
}

// GetUserByEmail retrieves a user by email in a transaction.
func (s *Service) GetUserByEmail(
	ctx context.Context,
	email string,
) (u *dto.UserRead, err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository((*userrepo.Repository)(nil))
		if err != nil {
			return err
		}
		repo, ok := repoAny.(userrepo.Repository)
		if !ok {
			return fmt.Errorf("unexpected repository type")
		}
		u, err = repo.GetByEmail(ctx, email)
		return err
	})
	if err != nil {
		u = nil
	}
	return
}

// GetUserByUsername retrieves a user by username in a transaction.
func (s *Service) GetUserByUsername(
	ctx context.Context,
	username string,
) (u *dto.UserRead, err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository((*userrepo.Repository)(nil))
		if err != nil {
			return err
		}
		repo, ok := repoAny.(userrepo.Repository)
		if !ok {
			return fmt.Errorf("unexpected repository type")
		}
		u, err = repo.GetByUsername(ctx, username)
		return err
	})
	if err != nil {
		u = nil
	}
	return
}

// UpdateUser updates user information in a transaction.
func (s *Service) UpdateUser(
	ctx context.Context,
	userID string,
	update *dto.UserUpdate,
) (err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository((*userrepo.Repository)(nil))
		if err != nil {
			return err
		}
		repo, ok := repoAny.(userrepo.Repository)
		if !ok {
			return fmt.Errorf("unexpected repository type")
		}
		uid, parseErr := uuid.Parse(userID)
		if parseErr != nil {
			return parseErr
		}
		u, err := repo.Get(ctx, uid)
		if err != nil {
			return err
		}
		if u == nil {
			return user.ErrUserNotFound
		}

		return repo.Update(ctx, uid, update)
	})
	return
}

// DeleteUser deletes a user account in a transaction.
func (s *Service) DeleteUser(
	ctx context.Context,
	userID string,
) (err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository((*userrepo.Repository)(nil))
		if err != nil {
			return err
		}
		repo, ok := repoAny.(userrepo.Repository)
		if !ok {
			return fmt.Errorf("unexpected repository type")
		}
		uid, parseErr := uuid.Parse(userID)
		if parseErr != nil {
			return parseErr
		}
		return repo.Delete(ctx, uid)
	})
	return
}

// ValidUser validates user credentials in a transaction.
func (s *Service) ValidUser(
	ctx context.Context,
	identifier string, // Can be either email or username
	password string,
) (
	valid bool,
	err error,
) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository((*userrepo.Repository)(nil))
		if err != nil {
			return err
		}
		repo, ok := repoAny.(userrepo.Repository)
		if !ok {
			return fmt.Errorf("unexpected repository type")
		}

		// Try to get u by email first
		u, err := repo.GetByEmail(ctx, identifier)
		if err != nil || u == nil {
			// If not found by email, try by username
			u, err = repo.GetByUsername(ctx, identifier)
			if err != nil || u == nil {
				// User not found by either email or username
				return nil
			}
		}

		// Check if the provided password matches the stored hash
		valid = utils.CheckPasswordHash(password, u.HashedPassword)
		return nil
	})
	return
}
