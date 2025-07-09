package service

import (
	"log/slog"

	"github.com/amirasaad/fintech/pkg/decorator"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

type UserService struct {
	uowFactory  func() (repository.UnitOfWork, error)
	logger      *slog.Logger
	transaction decorator.TransactionDecorator
}

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

func (s *UserService) CreateUser(
	username, email, password string,
) (u *user.User, err error) {
	s.logger.Info("CreateUser started", "username", username, "email", email)
	defer func() {
		if err != nil {
			s.logger.Error("CreateUser failed", "username", username, "email", email, "error", err)
		} else if u != nil {
			s.logger.Info("CreateUser successful", "username", username, "email", email, "userID", u.ID)
		}
	}()
	var uLocal *user.User
	err = s.transaction.Execute(func() error {
		uLocal, err = user.NewUser(username, email, password)
		if err != nil {
			logger := s.logger.With("username", username, "email", email)
			logger.Error("CreateUser failed: domain error", "error", err)
			return err
		}

		uow, err := s.uowFactory()
		if err != nil {
			logger := s.logger.With("username", username, "email", email)
			logger.Error("CreateUser failed: uowFactory error", "error", err)
			return err
		}

		repo, err := uow.UserRepository()
		if err != nil {
			logger := s.logger.With("username", username, "email", email)
			logger.Error("CreateUser failed: UserRepository error", "error", err)
			return err
		}

		err = repo.Create(uLocal)
		if err != nil {
			logger := s.logger.With("username", username, "email", email)
			logger.Error("CreateUser failed: repo create error", "error", err)
			return err
		}

		return nil
	})
	if err != nil {
		logger := s.logger.With("username", username, "email", email)
		logger.Error("CreateUser failed: transaction error", "error", err)
		return nil, err
	}
	u = uLocal
	logger := s.logger.With("username", username, "email", email)
	logger.Info("CreateUser successful", "userID", u.ID)
	return
}

func (s *UserService) GetUser(
	userID string,
) (u *user.User, err error) {
	s.logger.Info("GetUser started", "userID", userID)
	defer func() {
		if err != nil {
			s.logger.Error("GetUser failed", "userID", userID, "error", err)
		} else if u != nil {
			s.logger.Info("GetUser successful", "userID", userID, "foundUserID", u.ID)
		}
	}()
	uid, parseErr := uuid.Parse(userID)
	if parseErr != nil {
		err = parseErr
		return
	}
	uow, err := s.uowFactory()
	if err != nil {
		err = err
		return
	}
	repo, err := uow.UserRepository()
	if err != nil {
		err = err
		return
	}
	u, err = repo.Get(uid)
	if err != nil {
		u = nil
	}
	return
}

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
	uow, err := s.uowFactory()
	if err != nil {
		return
	}
	repo, err := uow.UserRepository()
	if err != nil {
		return
	}
	u, err := repo.Get(uid)
	if err != nil {
		return
	}
	if err = updateFn(u); err != nil {
		return
	}
	err = repo.Update(u)
	return
}

func (s *UserService) DeleteUser(
	userID string,
) (err error) {
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
	uow, err := s.uowFactory()
	if err != nil {
		return
	}
	repo, err := uow.UserRepository()
	if err != nil {
		return
	}
	err = repo.Delete(uid)
	return
}

func (s *UserService) ValidUser(
	id uuid.UUID,
	password string,
) (isValid bool, err error) {
	uow, err := s.uowFactory()
	if err != nil {
		return
	}

	repo, err := uow.UserRepository()
	if err != nil {
		return
	}

	isValid = repo.Valid(id, password)
	return
}
