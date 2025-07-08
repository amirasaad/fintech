package service

import (
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"

	"github.com/google/uuid"
)

type UserService struct {
	uowFactory func() (repository.UnitOfWork, error)
}

func NewUserService(
	uowFactory func() (repository.UnitOfWork, error),
) *UserService {
	return &UserService{
		uowFactory: uowFactory,
	}
}

func (s *UserService) CreateUser(
	username, email, password string,
) (u *domain.User, err error) {
	uow, err := s.uowFactory()
	if err != nil {
		u = nil
		return
	}
	err = uow.Begin()
	if err != nil {
		u = nil
		return
	}

	u, err = domain.NewUser(username, email, password)
	if err != nil {
		u = nil
		return
	}

	repo, err := uow.UserRepository()
	if err != nil {
		_ = uow.Rollback()
		u = nil
		return
	}

	err = repo.Create(u)
	if err != nil {
		_ = uow.Rollback()
		u = nil
		return
	}

	err = uow.Commit()
	if err != nil {
		_ = uow.Rollback()
		u = nil
		return
	}
	return
}

func (s *UserService) GetUser(id uuid.UUID) (u *domain.User, err error) {
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

func (s *UserService) GetUserByEmail(email string) (u *domain.User, err error) {
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

func (s *UserService) GetUserByUsername(username string) (u *domain.User, err error) {
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

func (s *UserService) UpdateUser(u *domain.User) (err error) {
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
