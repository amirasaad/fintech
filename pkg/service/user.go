package service

import (
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

type UserService struct {
	uowFactory func() (repository.UnitOfWork, error)
	logger     *slog.Logger
}

func NewUserService(
	uowFactory func() (repository.UnitOfWork, error),
	logger *slog.Logger,
) *UserService {
	return &UserService{
		uowFactory: uowFactory,
		logger:     logger,
	}
}

func (s *UserService) CreateUser(
	username, email, password string,
) (u *user.User, err error) {
	logger := s.logger.With("username", username, "email", email)
	uow, err := s.uowFactory()
	if err != nil {
		logger.Error("CreateUser failed: uowFactory error", "error", err)
		u = nil
		return
	}
	err = uow.Begin()
	if err != nil {
		logger.Error("CreateUser failed: begin error", "error", err)
		u = nil
		return
	}

	u, err = user.NewUser(username, email, password)
	if err != nil {
		logger.Error("CreateUser failed: domain error", "error", err)
		u = nil
		return
	}

	repo, err := uow.UserRepository()
	if err != nil {
		logger.Error("CreateUser failed: UserRepository error", "error", err)
		_ = uow.Rollback()
		u = nil
		return
	}

	err = repo.Create(u)
	if err != nil {
		logger.Error("CreateUser failed: repo create error", "error", err)
		_ = uow.Rollback()
		u = nil
		return
	}

	err = uow.Commit()
	if err != nil {
		logger.Error("CreateUser failed: commit error", "error", err)
		_ = uow.Rollback()
		u = nil
		return
	}
	logger.Info("CreateUser successful", "userID", u.ID)
	return
}

func (s *UserService) GetUser(id uuid.UUID) (u *user.User, err error) {
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

	u, err = repo.Get(id)
	if err != nil {
		u = nil
		return
	}

	return
}

func (s *UserService) GetUserByEmail(email string) (u *user.User, err error) {
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

func (s *UserService) GetUserByUsername(username string) (u *user.User, err error) {
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

func (s *UserService) UpdateUser(u *user.User) (err error) {
	uow, err := s.uowFactory()
	if err != nil {
		return
	}
	err = uow.Begin()
	if err != nil {
		return
	}

	repo, err := uow.UserRepository()
	if err != nil {
		_ = uow.Rollback()
		return
	}

	err = repo.Update(u)
	if err != nil {
		_ = uow.Rollback()
		return
	}

	err = uow.Commit()
	if err != nil {
		_ = uow.Rollback()
		return
	}
	return
}

func (s *UserService) DeleteUser(id uuid.UUID) (err error) {
	uow, err := s.uowFactory()
	if err != nil {
		return
	}
	err = uow.Begin()
	if err != nil {
		return
	}

	repo, err := uow.UserRepository()
	if err != nil {
		_ = uow.Rollback()
		return
	}

	err = repo.Delete(id)
	if err != nil {
		_ = uow.Rollback()
		return
	}

	err = uow.Commit()
	if err != nil {
		_ = uow.Rollback()
		return
	}
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
