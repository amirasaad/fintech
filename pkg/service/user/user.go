// Package user provides business logic for user management operations.
// It uses the decorator pattern for transaction management and includes comprehensive logging.
package user

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/config"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

// Service provides business logic for user operations including creation, updates, and deletion.
type Service struct {
	uow    repository.UnitOfWork
	logger *slog.Logger
}

// NewService creates a new Service with a UnitOfWork and logger.
func NewService(
	deps config.Deps,
) *Service {
	return &Service{
		uow:    deps.Uow,
		logger: deps.Logger,
	}
}

// CreateUser creates a new user account in a transaction.
func (s *Service) CreateUser(
	ctx context.Context,
	username, email, password string,
) (u *user.User, err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repo, err := uow.UserRepository()
		if err != nil {
			return err
		}
		u, err = user.NewUser(username, email, password)
		if err != nil {
			return err
		}
		return repo.Create(u)
	})
	if err != nil {
		u = nil
	}
	return
}

// GetUser retrieves a user by ID in a transaction.
func (s *Service) GetUser(ctx context.Context, userID string) (u *user.User, err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repo, err := uow.UserRepository()
		if err != nil {
			return err
		}
		uid, parseErr := uuid.Parse(userID)
		if parseErr != nil {
			return parseErr
		}
		u, err = repo.Get(uid)
		return err
	})
	if err != nil {
		u = nil
	}
	return
}

// GetUserByEmail retrieves a user by email in a transaction.
func (s *Service) GetUserByEmail(ctx context.Context, email string) (u *user.User, err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repo, err := uow.UserRepository()
		if err != nil {
			return err
		}
		u, err = repo.GetByEmail(email)
		return err
	})
	if err != nil {
		u = nil
	}
	return
}

// GetUserByUsername retrieves a user by username in a transaction.
func (s *Service) GetUserByUsername(ctx context.Context, username string) (u *user.User, err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repo, err := uow.UserRepository()
		if err != nil {
			return err
		}
		u, err = repo.GetByUsername(username)
		return err
	})
	if err != nil {
		u = nil
	}
	return
}

// UpdateUser updates user information in a transaction.
func (s *Service) UpdateUser(ctx context.Context, userID string, updateFn func(u *user.User) error) (err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repo, err := uow.UserRepository()
		if err != nil {
			return err
		}
		uid, parseErr := uuid.Parse(userID)
		if parseErr != nil {
			return parseErr
		}
		u, err := repo.Get(uid)
		if err != nil {
			return err
		}
		if u == nil {
			return user.ErrUserNotFound
		}
		if err = updateFn(u); err != nil {
			return err
		}
		return repo.Update(u)
	})
	return
}

// DeleteUser deletes a user account in a transaction.
func (s *Service) DeleteUser(ctx context.Context, userID string) (err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repo, err := uow.UserRepository()
		if err != nil {
			return err
		}
		uid, parseErr := uuid.Parse(userID)
		if parseErr != nil {
			return parseErr
		}
		return repo.Delete(uid)
	})
	return
}

// ValidUser validates user credentials in a transaction.
func (s *Service) ValidUser(ctx context.Context, userID string, password string) (valid bool, err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repo, err := uow.UserRepository()
		if err != nil {
			return err
		}
		uid, parseErr := uuid.Parse(userID)
		if parseErr != nil {
			return parseErr
		}
		valid = repo.Valid(uid, password)
		return nil
	})
	return
}
