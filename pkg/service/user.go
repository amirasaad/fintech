package service

import (
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"

	"github.com/google/uuid"
)

type UserService struct {
	uowFactory func() (repository.UnitOfWork, error)
}

func NewUserService(uowFactory func() (repository.UnitOfWork, error)) *UserService {
	return &UserService{
		uowFactory: uowFactory,
	}
}

func (s *UserService) CreateUser(username, email, password string) (*domain.User, error) {
	uow, err := s.uowFactory()
	if err != nil {
		return nil, err
	}
	err = uow.Begin()
	if err != nil {
		return nil, err
	}

	u, err := domain.NewUser(username, email, password)
	if err != nil {
		return nil, err
	}
	err = uow.UserRepository().Create(u)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	err = uow.Commit()
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}
	return u, nil
}

func (s *UserService) GetUser(id uuid.UUID) (*domain.User, error) {
	uow, err := s.uowFactory()
	if err != nil {
		return nil, err
	}

	u, err := uow.UserRepository().Get(id)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (s *UserService) GetUserByEmail(email string) (*domain.User, error) {
	uow, err := s.uowFactory()
	if err != nil {
		return nil, err
	}

	u, err := uow.UserRepository().GetByEmail(email)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *UserService) GetUserByUsername(username string) (*domain.User, error) {
	uow, err := s.uowFactory()
	if err != nil {
		return nil, err
	}

	u, err := uow.UserRepository().GetByUsername(username)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *UserService) UpdateUser(u *domain.User) error {
	uow, err := s.uowFactory()
	if err != nil {
		return err
	}
	err = uow.Begin()
	if err != nil {
		return err
	}

	err = uow.UserRepository().Update(u)
	if err != nil {
		_ = uow.Rollback()
		return err
	}

	err = uow.Commit()
	if err != nil {
		_ = uow.Rollback()
		return err
	}
	return nil
}

func (s *UserService) DeleteUser(id uuid.UUID) error {
	uow, err := s.uowFactory()
	if err != nil {
		return err
	}
	err = uow.Begin()
	if err != nil {
		return err
	}

	err = uow.UserRepository().Delete(id)
	if err != nil {
		_ = uow.Rollback()
		return err
	}

	err = uow.Commit()
	if err != nil {
		_ = uow.Rollback()
		return err
	}
	return nil
}

func (s *UserService) ValidUser(id uuid.UUID, password string) bool {
	uow, err := s.uowFactory()
	if err != nil {
		return false
	}

	return uow.UserRepository().Valid(id, password)
}
