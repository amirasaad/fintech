package handler

import (
	"bytes"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/amirasaad/fintech/internal/domain"
	"github.com/amirasaad/fintech/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/gofiber/fiber/v2"
)

type AccountMockRepo struct {
	mock.Mock
}

func (m *AccountMockRepo) Create(account *domain.Account) error {
	args := m.Called(account)
	return args.Error(0)
}
func (m *AccountMockRepo) Get(id uuid.UUID) (*domain.Account, error) {
	args := m.Called(id)
	return args.Get(0).(*domain.Account), args.Error(1)
}
func (m *AccountMockRepo) Update(account *domain.Account) error {
	args := m.Called(account)
	return args.Error(0)
}
func (m *AccountMockRepo) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

type TransactionMockRepo struct {
	mock.Mock
}

func (m *TransactionMockRepo) Create(transaction *domain.Transaction) error {
	args := m.Called(transaction)
	return args.Error(0)
}
func (m *TransactionMockRepo) Get(id uuid.UUID) (*domain.Transaction, error) {
	args := m.Called(id)
	return args.Get(0).(*domain.Transaction), args.Error(1)
}
func (m *TransactionMockRepo) List(accountID uuid.UUID) ([]*domain.Transaction, error) {
	args := m.Called(accountID)
	return args.Get(0).([]*domain.Transaction), args.Error(1)
}

type MockUoW struct {
	Account         *AccountMockRepo
	TransactionRepo *TransactionMockRepo
}

func (m *MockUoW) Begin() error {
	return nil
}
func (m *MockUoW) Commit() error {
	return nil
}
func (m *MockUoW) Rollback() error {
	return nil
}
func (m *MockUoW) AccountRepository() repository.AccountRepository {
	return m.Account
}
func (m *MockUoW) TransactionRepository() repository.TransactionRepository {
	return m.TransactionRepo
}

func TestAccountRoutes(t *testing.T) {
	app := fiber.New()
	accountRepo := &AccountMockRepo{}
	transactionRepo := &TransactionMockRepo{}
	mockUow := &MockUoW{Account: accountRepo, TransactionRepo: transactionRepo}
	AccountRoutes(app, func() (repository.UnitOfWork, error) { return mockUow, nil })

	accountRepo.On("Create", mock.Anything).Return(nil)
	// Test the route
	req := httptest.NewRequest("POST", "/account", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	accountRepo.On("Get", mock.Anything).Return(domain.NewAccount(), nil)
	transactionRepo.On("Create", mock.Anything).Return(nil)
	accountRepo.On("Update", mock.Anything).Return(nil)
	depositBody := bytes.NewBuffer([]byte(`{"amount": 100.0}`))
	req = httptest.NewRequest("POST", fmt.Sprintf("/account/%s/deposit", uuid.New()), depositBody)
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	withdrawBody := bytes.NewBuffer([]byte(`{"amount": 100.0}`))
	req = httptest.NewRequest("POST", fmt.Sprintf("/account/%s/withdraw", uuid.New()), withdrawBody)
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}
}

func TestAccountRoutesFailureAccountNotFound(t *testing.T) {
	app := fiber.New()
	accountRepo := &AccountMockRepo{}
	transactionRepo := &TransactionMockRepo{}
	mockUow := &MockUoW{Account: accountRepo, TransactionRepo: transactionRepo}
	AccountRoutes(app, func() (repository.UnitOfWork, error) { return mockUow, nil })

	accountRepo.On("Get", mock.Anything).Return(&domain.Account{}, errors.New("account not found"))

	// Test the route
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/deposit", uuid.New()), bytes.NewBuffer([]byte(`{"amount": 100.0}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("Expected status %d, got %d", fiber.StatusNotFound, resp.StatusCode)
	}
}
func TestAccountRoutesFailureTransaction(t *testing.T) {
	app := fiber.New()
	accountRepo := &AccountMockRepo{}
	transactionRepo := &TransactionMockRepo{}
	mockUow := &MockUoW{Account: accountRepo, TransactionRepo: transactionRepo}
	AccountRoutes(app, func() (repository.UnitOfWork, error) { return mockUow, nil })

	accountRepo.On("Get", mock.Anything).Return(&domain.Account{Balance: 100.0}, nil)

	// test deposit negative amount
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/deposit", uuid.New()), bytes.NewBuffer([]byte(`{"amount": -100.0}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", fiber.StatusBadRequest, resp.StatusCode)
	}

	// test withdraw negative amount
	req = httptest.NewRequest("POST", fmt.Sprintf("/account/%s/withdraw", uuid.New()), bytes.NewBuffer([]byte(`{"amount": -100.0}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", fiber.StatusBadRequest, resp.StatusCode)
	}

	// test withdraw amount greater than balance
	req = httptest.NewRequest("POST", fmt.Sprintf("/account/%s/withdraw", uuid.New()), bytes.NewBuffer([]byte(`{"amount": 1000.0}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", fiber.StatusBadRequest, resp.StatusCode)
	}
}
