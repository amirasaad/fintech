package service

import (
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

type AccountService struct {
	uowFactory func() (repository.UnitOfWork, error)
}

func NewAccountService(uowFactory func() (repository.UnitOfWork, error)) *AccountService {
	return &AccountService{
		uowFactory: uowFactory,
	}
}

func (s *AccountService) CreateAccount() (*domain.Account, error) {
	uow, err := s.uowFactory()
	if err != nil {
		return nil, err
	}
	err = uow.Begin()
	if err != nil {
		return nil, err
	}

	a := domain.NewAccount()
	err = uow.AccountRepository().Create(a)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	err = uow.Commit()
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (s *AccountService) Deposit(accountID uuid.UUID, amount float64) (*domain.Transaction, error) {
	uow, err := s.uowFactory()
	if err != nil {
		return nil, err
	}
	err = uow.Begin()
	if err != nil {
		return nil, err
	}

	a, err := uow.AccountRepository().Get(accountID)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	tx, err := a.Deposit(amount)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	err = uow.AccountRepository().Update(a)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	err = uow.TransactionRepository().Create(tx)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	err = uow.Commit()
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (s *AccountService) Withdraw(accountID uuid.UUID, amount float64) (*domain.Transaction, error) {
	uow, err := s.uowFactory()
	if err != nil {
		return nil, err
	}
	err = uow.Begin()
	if err != nil {
		return nil, err
	}

	a, err := uow.AccountRepository().Get(accountID)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	tx, err := a.Withdraw(amount)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	err = uow.AccountRepository().Update(a)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	err = uow.TransactionRepository().Create(tx)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	err = uow.Commit()
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (s *AccountService) GetAccount(accountID uuid.UUID) (*domain.Account, error) {
	uow, err := s.uowFactory()
	if err != nil {
		return nil, err
	}
	err = uow.Begin()
	if err != nil {
		return nil, err
	}

	a, err := uow.AccountRepository().Get(accountID)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	err = uow.Commit()
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (s *AccountService) GetTransactions(accountID uuid.UUID) ([]*domain.Transaction, error) {
	uow, err := s.uowFactory()
	if err != nil {
		return nil, err
	}
	txs, err := uow.TransactionRepository().List(accountID)
	if err != nil {
		_ = uow.Rollback()
		return nil, err
	}

	return txs, nil
}

func (s *AccountService) GetBalance(accountID uuid.UUID) (float64, error) {
	uow, err := s.uowFactory()
	if err != nil {
		return 0, err
	}
	a, err := uow.AccountRepository().Get(accountID)
	if err != nil {
		_ = uow.Rollback()
		return 0, err
	}

	return float64(a.Balance) / 100, nil
}
