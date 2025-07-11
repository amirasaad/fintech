// Package service provides business logic for user management operations.
// It uses the decorator pattern for transaction management and includes comprehensive logging.
package service

import (
	"context"
	"log/slog"
	"reflect"

	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

// UserService provides business logic for user operations including creation, updates, and deletion.
type UserService struct {
	uow    repository.UnitOfWork
	logger *slog.Logger
}

// NewUserService creates a new UserService with a UnitOfWork and logger.
func NewUserService(
	uow repository.UnitOfWork,
	logger *slog.Logger,
) *UserService {
	return &UserService{
		uow:    uow,
		logger: logger,
	}
}

// CreateUser creates a new user account in a transaction.
func (s *UserService) CreateUser(
	ctx context.Context,
	username, email, password string,
) (u *user.User, err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.UserRepository)(nil)).Elem())
		if err != nil {
			return err
		}
		repo := repoAny.(repository.UserRepository)
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
func (s *UserService) GetUser(ctx context.Context, userID string) (u *user.User, err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.UserRepository)(nil)).Elem())
		if err != nil {
			return err
		}
		repo := repoAny.(repository.UserRepository)
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
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (u *user.User, err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.UserRepository)(nil)).Elem())
		if err != nil {
			return err
		}
		repo := repoAny.(repository.UserRepository)
		u, err = repo.GetByEmail(email)
		return err
	})
	if err != nil {
		u = nil
	}
	return
}

// GetUserByUsername retrieves a user by username in a transaction.
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (u *user.User, err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.UserRepository)(nil)).Elem())
		if err != nil {
			return err
		}
		repo := repoAny.(repository.UserRepository)
		u, err = repo.GetByUsername(username)
		return err
	})
	if err != nil {
		u = nil
	}
	return
}

// UpdateUser updates user information in a transaction.
func (s *UserService) UpdateUser(ctx context.Context, userID string, updateFn func(u *user.User) error) (err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.UserRepository)(nil)).Elem())
		if err != nil {
			return err
		}
		repo := repoAny.(repository.UserRepository)
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
func (s *UserService) DeleteUser(ctx context.Context, userID string) (err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.UserRepository)(nil)).Elem())
		if err != nil {
			return err
		}
		repo := repoAny.(repository.UserRepository)
		uid, parseErr := uuid.Parse(userID)
		if parseErr != nil {
			return parseErr
		}
		return repo.Delete(uid)
	})
	return
}

// ValidUser validates user credentials in a transaction.
func (s *UserService) ValidUser(ctx context.Context, userID string, password string) (valid bool, err error) {
	err = s.uow.Do(ctx, func(uow repository.UnitOfWork) error {
		repoAny, err := uow.GetRepository(reflect.TypeOf((*repository.UserRepository)(nil)).Elem())
		if err != nil {
			return err
		}
		repo := repoAny.(repository.UserRepository)
		uid, parseErr := uuid.Parse(userID)
		if parseErr != nil {
			return parseErr
		}
		valid = repo.Valid(uid, password)
		return nil
	})
	return
}
