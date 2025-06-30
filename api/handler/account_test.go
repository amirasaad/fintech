package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

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
	mock.Mock
	AccountRepo     *AccountMockRepo
	TransactionRepo *TransactionMockRepo
}

func (m *MockUoW) Begin() error {
	return nil
}
func (m *MockUoW) Commit() error {
	return nil
}
func (m *MockUoW) Rollback() error {
	args := m.Called()
	return args.Error(0)
}
func (m *MockUoW) AccountRepository() repository.AccountRepository {
	return m.AccountRepo
}
func (m *MockUoW) TransactionRepository() repository.TransactionRepository {
	return m.TransactionRepo
}

func TestAccountRoutes(t *testing.T) {
	app := fiber.New()
	accountRepo := &AccountMockRepo{}
	transactionRepo := &TransactionMockRepo{}
	mockUow := &MockUoW{AccountRepo: accountRepo, TransactionRepo: transactionRepo}
	AccountRoutes(app, func() (repository.UnitOfWork, error) { return mockUow, nil })

	accountRepo.On("Create", mock.Anything).Return(nil)
	// Test the route
	req := httptest.NewRequest("POST", "/account", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck
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
	defer resp.Body.Close() //nolint:errcheck
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
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}
}

func TestAccountRoutesFailureAccountNotFound(t *testing.T) {
	app := fiber.New()
	accountRepo := &AccountMockRepo{}
	transactionRepo := &TransactionMockRepo{}
	mockUow := &MockUoW{AccountRepo: accountRepo, TransactionRepo: transactionRepo}
	mockUow.On("Rollback").Return(nil)
	AccountRoutes(app, func() (repository.UnitOfWork, error) { return mockUow, nil })

	accountRepo.On("Get", mock.Anything).Return(&domain.Account{}, errors.New("account not found"))

	// Test the route
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/deposit", uuid.New()), bytes.NewBuffer([]byte(`{"amount": 100.0}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", fiber.StatusNotFound, resp.StatusCode)
	}
	mockUow.AssertCalled(t, "Rollback")
}
func TestAccountRoutesFailureTransaction(t *testing.T) {
	app := fiber.New()
	accountRepo := &AccountMockRepo{}
	transactionRepo := &TransactionMockRepo{}
	mockUow := &MockUoW{AccountRepo: accountRepo, TransactionRepo: transactionRepo}
	mockUow.On("Rollback").Return(nil)
	AccountRoutes(app, func() (repository.UnitOfWork, error) { return mockUow, nil })

	accountRepo.On("Get", mock.Anything).Return(&domain.Account{Balance: 100.0}, nil)

	// test deposit negative amount
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/deposit", uuid.New()), bytes.NewBuffer([]byte(`{"amount": -100.0}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", fiber.StatusUnprocessableEntity, resp.StatusCode)
	}

	// test withdraw negative amount
	req = httptest.NewRequest("POST", fmt.Sprintf("/account/%s/withdraw", uuid.New()), bytes.NewBuffer([]byte(`{"amount": -100.0}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck
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
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d", fiber.StatusBadRequest, resp.StatusCode)
	}
}

func TestAccountRoutesTransactionList(t *testing.T) {
	assert := assert.New(t)
	app := fiber.New()
	accountRepo := &AccountMockRepo{}
	transactionRepo := &TransactionMockRepo{}
	mockUow := &MockUoW{AccountRepo: accountRepo, TransactionRepo: transactionRepo}
	AccountRoutes(app, func() (repository.UnitOfWork, error) { return mockUow, nil })

	accountID := uuid.New()
	accountRepo.On("Get", accountID).Return(&domain.Account{ID: accountID}, nil)
	created1, _ := time.Parse(time.RFC3339, "2023-10-01T00:00:00Z")
	created2, _ := time.Parse(time.RFC3339, "2023-10-02T00:00:00Z")
	transactionRepo.On("List", accountID).Return([]*domain.Transaction{
		{ID: uuid.New(), Amount: 100.0, Balance: 100.0, Created: created1},
		{ID: uuid.New(), Amount: 50.0, Balance: 150.0, Created: created2},
	}, nil)

	req := httptest.NewRequest("GET", fmt.Sprintf("/account/%s/transactions", accountID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	var transactions []domain.Transaction
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	if err := json.Unmarshal(bodyBytes, &transactions); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	assert.Equal(2, len(transactions), "Expected 2 transactions")
}

func TestAccountRoutesBalance(t *testing.T) {
	assert := assert.New(t)
	app := fiber.New()
	accountRepo := &AccountMockRepo{}
	transactionRepo := &TransactionMockRepo{}
	mockUow := &MockUoW{AccountRepo: accountRepo, TransactionRepo: transactionRepo}
	AccountRoutes(app, func() (repository.UnitOfWork, error) { return mockUow, nil })

	accountID := uuid.New()
	account := &domain.Account{ID: accountID, Balance: 100.0}
	accountRepo.On("Get", accountID).Return(account, nil)

	req := httptest.NewRequest("GET", fmt.Sprintf("/account/%s/balance", accountID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck
	assert.Equal(fiber.StatusOK, resp.StatusCode)

	var balanceResponse struct {
		Balance float64 `json:"balance"`
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	if err := json.Unmarshal(bodyBytes, &balanceResponse); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	assert.Equal(float64(account.Balance/100.0), balanceResponse.Balance, "Expected balance to match")
}

func TestAccountRoutesUoWError(t *testing.T) {
	app := fiber.New()
	AccountRoutes(app, func() (repository.UnitOfWork, error) { return nil, errors.New("unit of work error") })

	// Test the route
	req := httptest.NewRequest("POST", "/account", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", fiber.StatusInternalServerError, resp.StatusCode)
	}

	req = httptest.NewRequest("GET", fmt.Sprintf("/account/%s/balance", uuid.New()), nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", fiber.StatusInternalServerError, resp.StatusCode)
	}

	req = httptest.NewRequest("GET", fmt.Sprintf("/account/%s/transactions", uuid.New()), nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", fiber.StatusInternalServerError, resp.StatusCode)
	}
	req = httptest.NewRequest("POST", fmt.Sprintf("/account/%s/deposit", uuid.New()), bytes.NewBuffer([]byte(`{"amount": 100.0}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", fiber.StatusInternalServerError, resp.StatusCode)
	}

	req = httptest.NewRequest("POST", fmt.Sprintf("/account/%s/withdraw", uuid.New()), bytes.NewBuffer([]byte(`{"amount": 100.0}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", fiber.StatusInternalServerError, resp.StatusCode)
	}
}

func TestAccountRoutesRollbackWhenCreateFails(t *testing.T) {
	app := fiber.New()
	accountRepo := &AccountMockRepo{}
	transactionRepo := &TransactionMockRepo{}
	mockUow := &MockUoW{AccountRepo: accountRepo, TransactionRepo: transactionRepo}
	mockUow.On("Rollback").Return(nil)
	AccountRoutes(app, func() (repository.UnitOfWork, error) { return mockUow, nil })

	accountRepo.On("Create", mock.Anything).Return(errors.New("failed to create account"))

	// Test the route
	req := httptest.NewRequest("POST", "/account", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	mockUow.AssertCalled(t, "Rollback")
}

func TestAccountRoutesRollbackWhenDepositFails(t *testing.T) {
	app := fiber.New()
	accountRepo := &AccountMockRepo{}
	transactionRepo := &TransactionMockRepo{}
	mockUow := &MockUoW{AccountRepo: accountRepo, TransactionRepo: transactionRepo}
	mockUow.On("Rollback").Return(nil)

	AccountRoutes(app, func() (repository.UnitOfWork, error) { return mockUow, nil })

	accountID := uuid.New()
	accountRepo.On("Get", accountID).Return(&domain.Account{ID: accountID}, nil)
	transactionRepo.On("Create", mock.Anything).Return(errors.New("failed to create transaction"))
	accountRepo.On("Update", mock.Anything).Return(nil)
	// Test the route
	depositBody := bytes.NewBuffer([]byte(`{"amount": 100.0}`))
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/deposit", accountID), depositBody)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	mockUow.AssertCalled(t, "Rollback")
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", fiber.StatusInternalServerError, resp.StatusCode)
	}
}

func TestSimultaneousRequests(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	app := fiber.New()
	accountRepo := &AccountMockRepo{}
	transactionRepo := &TransactionMockRepo{}
	mockUow := &MockUoW{AccountRepo: accountRepo, TransactionRepo: transactionRepo}
	AccountRoutes(app, func() (repository.UnitOfWork, error) { return mockUow, nil })
	initialBalance := 1000.0
	acc := domain.NewAccount()
	_, err := acc.Deposit(initialBalance)
	require.NoError(err, "Initial deposit should not return an error")

	// Remove testify's On/Return for Get/Update, use only our mutex-protected methods
	accountRepo.On("Get", mock.Anything).Return(acc, nil)
	accountRepo.On("Update", mock.Anything).Return(nil)
	transactionRepo.On("Create", mock.Anything).Return(nil)

	numOperations := 1000
	depositAmount := 10.0
	withdrawAmount := 5.0

	var wg sync.WaitGroup
	wg.Add(numOperations * 2)

	for range numOperations {
		go func() {
			defer wg.Done()
			depositBody := bytes.NewBuffer(fmt.Appendf(nil, `{"amount": %f}`, depositAmount))
			req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/deposit", uuid.New()), depositBody)
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			if err != nil {
				t.Errorf("Deposit request failed: %v", err)
				return
			}
			defer resp.Body.Close() //nolint:errcheck
			if resp.StatusCode != fiber.StatusOK {
				t.Errorf("Expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
			}
		}()
		go func() {
			defer wg.Done()
			withdrawBody := bytes.NewBuffer(fmt.Appendf(nil, `{"amount": %f}`, withdrawAmount))
			req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/withdraw", uuid.New()), withdrawBody)
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			if err != nil {
				t.Errorf("Withdraw request failed: %v", err)
				return
			}
			defer resp.Body.Close() //nolint:errcheck
			if resp.StatusCode != fiber.StatusOK {
				t.Errorf("Expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
			}
		}()
	}
	wg.Wait()

	// Check final balance
	// The expected balance is (numOperations * depositAmount) - (numOperations * withdrawAmount)
	req := httptest.NewRequest("GET", fmt.Sprintf("/account/%s/balance", acc.ID), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}
	var balanceResponse struct {
		Balance float64 `json:"balance"`
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	if err := json.Unmarshal(bodyBytes, &balanceResponse); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	expectedBalance := float64(initialBalance) + (float64(numOperations) * depositAmount) - (float64(numOperations) * withdrawAmount)
	assert.InDelta(expectedBalance, balanceResponse.Balance, 0.01)
}
