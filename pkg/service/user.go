// Package service provides business logic for user management operations.
// It uses the decorator pattern for transaction management and includes comprehensive logging.
package service

import (
	"log/slog"

	"github.com/amirasaad/fintech/pkg/decorator"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

// UserService provides business logic for user operations including creation, updates, and deletion.
// Uses the decorator pattern for automatic transaction management and structured logging.
type UserService struct {
	uowFactory  func() (repository.UnitOfWork, error)
	logger      *slog.Logger
	transaction decorator.TransactionDecorator
}

// NewUserService creates a new UserService with transaction decorator and logging.
func NewUserService(
	uowFactory func() (repository.UnitOfWork, error),
	logger *slog.Logger,
) *UserService {
	return &UserService{
		uowFactory:  uowFactory,
		logger:      logger,
		transaction: decorator.NewUnitOfWorkTransactionDecorator(uowFactory, logger),
	}
}

// CreateUser creates a new user account with automatic transaction management.
// Returns the created user or an error if the operation fails.
func (s *UserService) CreateUser(
	username, email, password string,
) (u *user.User, err error) {
	s.logger.Info("CreateUser started", "username", username, "email", email)
	defer func() {
		if err != nil {
			s.logger.Error("CreateUser failed", "username", username, "email", email, "error", err)
		} else {
			s.logger.Info("CreateUser successful", "username", username, "email", email, "userID", u.ID)
		}
	}()
	var uLocal *user.User
	err = s.transaction.Execute(func() error {
		uLocal, err = user.NewUser(username, email, password)
		if err != nil {
			return err
		}
		uow, err := s.uowFactory()
		if err != nil {
			return err
		}
		repo, err := uow.UserRepository()
		if err != nil {
			return err
		}
		err = repo.Create(uLocal)
		return err
	})
	if err != nil {
		s.logger.Error("CreateUser failed: transaction error", "username", username, "email", email, "error", err)
		return nil, err
	}
	u = uLocal
	return
}

// GetUser retrieves a user by ID with automatic transaction management.
// Returns the user or an error if not found.
func (s *UserService) GetUser(userID string) (u *user.User, err error) {
	s.logger.Info("GetUser started", "userID", userID)
	defer func() {
		if err != nil {
			s.logger.Error("GetUser failed", "userID", userID, "error", err)
		} else {
			s.logger.Info("GetUser successful", "userID", userID, "foundUserID", u.ID)
		}
	}()
	uid, parseErr := uuid.Parse(userID)
	if parseErr != nil {
		err = parseErr
		return
	}

	var uLocal *user.User
	uow, err := s.uowFactory()
	if err != nil {
		return
	}
	repo, err := uow.UserRepository()
	if err != nil {
		return
	}
	uLocal, err = repo.Get(uid)

	if err != nil {
		s.logger.Error("GetUser failed: transaction error", "userID", userID, "error", err)
		return
	}
	u = uLocal
	return
}

// GetUserByEmail retrieves a user by email with automatic transaction management.
// Returns the user or an error if not found.
func (s *UserService) GetUserByEmail(
	email string,
) (u *user.User, err error) {
	s.logger.Info("GetUserByEmail started", "email", email)
	defer func() {
		if err != nil {
			s.logger.Error("GetUserByEmail failed", "email", email, "error", err)
		} else if u != nil {
			s.logger.Info("GetUserByEmail successful", "email", email, "userID", u.ID)
		}
	}()
	uow, err := s.uowFactory()
	if err != nil {
		u = nil
		return
	}

	repo, err := uow.UserRepository()
	if err != nil {
		u = nil
		return
	}

	u, err = repo.GetByEmail(email)
	if err != nil {
		u = nil
		return
	}
	return
}

// GetUserByUsername retrieves a user by username with automatic transaction management.
// Returns the user or an error if not found.
func (s *UserService) GetUserByUsername(
	username string,
) (u *user.User, err error) {
	s.logger.Info("GetUserByUsername started", "username", username)
	defer func() {
		if err != nil {
			s.logger.Error("GetUserByUsername failed", "username", username, "error", err)
		} else if u != nil {
			s.logger.Info("GetUserByUsername successful", "username", username, "userID", u.ID)
		}
	}()
	uow, err := s.uowFactory()
	if err != nil {
		u = nil
		return
	}

	repo, err := uow.UserRepository()
	if err != nil {
		u = nil
		return
	}

	u, err = repo.GetByUsername(username)
	if err != nil {
		u = nil
		return
	}
	return
}

// UpdateUser updates user information with automatic transaction management.
// Returns an error if the user is not found or the operation fails.
func (s *UserService) UpdateUser(
	userID string,
	updateFn func(u *user.User) error,
) (err error) {
	s.logger.Info("UpdateUser started", "userID", userID)
	defer func() {
		if err != nil {
			s.logger.Error("UpdateUser failed", "userID", userID, "error", err)
		} else {
			s.logger.Info("UpdateUser successful", "userID", userID)
		}
	}()
	uid, parseErr := uuid.Parse(userID)
	if parseErr != nil {
		err = parseErr
		return
	}
	err = s.transaction.Execute(func() error {
		uow, err := s.uowFactory()
		if err != nil {
			return err
		}
		repo, err := uow.UserRepository()
		if err != nil {
			return err
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
		err = repo.Update(u)
		return err
	})
	return
}

// DeleteUser deletes a user account with automatic transaction management.
// Returns an error if the operation fails.
func (s *UserService) DeleteUser(userID string) (err error) {
	s.logger.Info("DeleteUser started", "userID", userID)
	defer func() {
		if err != nil {
			s.logger.Error("DeleteUser failed", "userID", userID, "error", err)
		} else {
			s.logger.Info("DeleteUser successful", "userID", userID)
		}
	}()
	uid, parseErr := uuid.Parse(userID)
	if parseErr != nil {
		err = parseErr
		return
	}
	err = s.transaction.Execute(func() error {
		uow, err := s.uowFactory()
		if err != nil {
			return err
		}
		repo, err := uow.UserRepository()
		if err != nil {
			return err
		}
		err = repo.Delete(uid)
		return err
	})
	return
}

// ValidUser validates user credentials with automatic transaction management.
// Returns true if credentials are valid, false otherwise.
func (s *UserService) ValidUser(userID string, password string) (valid bool, err error) {
	s.logger.Info("ValidUser started", "userID", userID)
	defer func() {
		if err != nil {
			s.logger.Error("ValidUser failed", "userID", userID, "error", err)
		} else {
			s.logger.Info("ValidUser completed", "userID", userID, "valid", valid)
		}
	}()
	uid, parseErr := uuid.Parse(userID)
	if parseErr != nil {
		err = parseErr
		return
	}
	var validLocal bool
	uow, err := s.uowFactory()
	if err != nil {
		return
	}
	repo, err := uow.UserRepository()
	if err != nil {
		return
	}
	validLocal = repo.Valid(uid, password)
	if err != nil {
		s.logger.Error("ValidUser failed: transaction error", "userID", userID, "error", err)
		return false, err
	}
	valid = validLocal
	return
}
