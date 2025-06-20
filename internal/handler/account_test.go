package handler

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/amirasaad/fintech/internal/domain"
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
func TestAccountRoutes(t *testing.T) {
	app := fiber.New()
	accountRepo := &AccountMockRepo{}
	transactionRepo := &TransactionMockRepo{}
	AccountRoutes(app, accountRepo, transactionRepo)

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
