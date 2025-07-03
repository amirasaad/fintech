package webapi

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
func (m *TransactionMockRepo) List(userID, accountID uuid.UUID) ([]*domain.Transaction, error) {
	args := m.Called(userID, accountID)
	return args.Get(0).([]*domain.Transaction), args.Error(1)
}

type UserMockRepo struct {
	mock.Mock
}

// Valid implements repository.UserRepository.
func (u *UserMockRepo) Valid(id uuid.UUID, password string) bool {
	args := u.Called(id, password)
	return args.Bool(0)
}

// Create implements repository.UserRepository.
func (u *UserMockRepo) Create(user *domain.User) error {
	args := u.Called(user)
	return args.Error(0)
}

// Delete implements repository.UserRepository.
func (u *UserMockRepo) Delete(id uuid.UUID) error {
	args := u.Called(id)
	return args.Error(0)
}

// Get implements repository.UserRepository.
func (u *UserMockRepo) Get(id uuid.UUID) (*domain.User, error) {
	args := u.Called(id)
	return args.Get(0).(*domain.User), args.Error(1)
}

// GetByEmail implements repository.UserRepository.
func (u *UserMockRepo) GetByEmail(email string) (*domain.User, error) {
	args := u.Called(email)
	return args.Get(0).(*domain.User), args.Error(1)
}

// GetByUsername implements repository.UserRepository.
func (u *UserMockRepo) GetByUsername(username string) (*domain.User, error) {
	args := u.Called(username)
	return args.Get(0).(*domain.User), args.Error(1)
}

// Update implements repository.UserRepository.
func (u *UserMockRepo) Update(user *domain.User) error {
	args := u.Called(user)
	return args.Error(0)
}

type MockUoW struct {
	mock.Mock
	UserRepo        *UserMockRepo
	AccountRepo     *AccountMockRepo
	TransactionRepo *TransactionMockRepo
}

func (m *MockUoW) Begin() error {
	args := m.Called()
	return args.Error(0)
}
func (m *MockUoW) Commit() error {
	args := m.Called()
	return args.Error(0)
}
func (m *MockUoW) Rollback() error {
	args := m.Called()
	return args.Error(0)
}

// UserRepository implements repository.UnitOfWork.
func (m *MockUoW) UserRepository() repository.UserRepository {
	return m.UserRepo
}
func (m *MockUoW) AccountRepository() repository.AccountRepository {
	return m.AccountRepo
}
func (m *MockUoW) TransactionRepository() repository.TransactionRepository {
	return m.TransactionRepo
}

func setupCommonMocks(userRepo *UserMockRepo, mockUow *MockUoW, testUser *domain.User) {
	userRepo.On("GetByUsername", "testuser").Return(testUser, nil)
	// userRepo.On("GetByEmail", "test@example.com").Return(testUser, nil)
	mockUow.On("Rollback").Return(nil)
}

func setupTestApp(t *testing.T) (*fiber.App, *UserMockRepo, *AccountMockRepo, *TransactionMockRepo, *MockUoW, *domain.User) {
	t.Helper()
	app := fiber.New()
	// app.Use(middleware.Protected())
	accountRepo := &AccountMockRepo{}
	transactionRepo := &TransactionMockRepo{}
	userRepo := &UserMockRepo{}
	mockUow := &MockUoW{UserRepo: userRepo, AccountRepo: accountRepo, TransactionRepo: transactionRepo}
	AuthRoutes(app, func() (repository.UnitOfWork, error) { return mockUow, nil })
	UserRoutes(app, func() (repository.UnitOfWork, error) { return mockUow, nil })
	AccountRoutes(app, func() (repository.UnitOfWork, error) { return mockUow, nil })
	testUser, _ := domain.NewUser("testuser", "testuser@example.com", "password123")

	return app, userRepo, accountRepo, transactionRepo, mockUow, testUser
}
func TestAccountCreate(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	app, userRepo, accountRepo, transactionRepo, mockUow, testUser := setupTestApp(t)
	accountRepo.On("Create", mock.Anything).Return(nil)

	mockUow.On("Begin").Return(nil)
	mockUow.On("Commit").Return(nil)

	req := httptest.NewRequest("POST", "/account", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))

	resp, err := app.Test(req)
	require.NoError(err)
	defer resp.Body.Close() //nolint:errcheck
	assert.Equal(fiber.StatusOK, resp.StatusCode)

	mockUow.AssertExpectations(t)
	userRepo.AssertExpectations(t)
	accountRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)
}

func TestAccountDeposit(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	app, userRepo, accountRepo, transactionRepo, mockUow, testUser := setupTestApp(t)

	accountRepo.On("Get", mock.Anything).Return(domain.NewAccount(testUser.ID), nil)
	transactionRepo.On("Create", mock.Anything).Return(nil)
	accountRepo.On("Update", mock.Anything).Return(nil)
	mockUow.On("Begin").Return(nil)
	mockUow.On("Commit").Return(nil)
	mockUow.On("Rollback").Return(nil)

	depositBody := bytes.NewBuffer([]byte(`{"amount": 100.0}`))
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/deposit", uuid.New()), depositBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))

	resp, err := app.Test(req)
	require.NoError(err)
	defer resp.Body.Close() //nolint:errcheck
	assert.Equal(fiber.StatusOK, resp.StatusCode)

	mockUow.AssertExpectations(t)
	userRepo.AssertExpectations(t)
	accountRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)
}

func TestAccountWithdraw(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	app, userRepo, accountRepo, transactionRepo, mockUow, testUser := setupTestApp(t)

	testAccount := domain.NewAccount(testUser.ID)
	testAccount.Deposit(testUser.ID, 1000)

	accountRepo.On("Get", mock.Anything).Return(testAccount, nil)
	transactionRepo.On("Create", mock.Anything).Return(nil)
	accountRepo.On("Update", mock.Anything).Return(nil)
	mockUow.On("Begin").Return(nil)
	mockUow.On("Commit").Return(nil)
	mockUow.On("Rollback").Return(nil)

	withdrawBody := bytes.NewBuffer([]byte(`{"amount": 100.0}`))
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/withdraw", uuid.New()), withdrawBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))

	resp, err := app.Test(req)
	require.NoError(err)
	defer resp.Body.Close() //nolint:errcheck
	assert.Equal(fiber.StatusOK, resp.StatusCode)

	mockUow.AssertExpectations(t)
	userRepo.AssertExpectations(t)
	accountRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)
}

func TestAccountRoutesFailureAccountNotFound(t *testing.T) {
	assert := assert.New(t)
	app, userRepo, accountRepo, transactionRepo, mockUow, testUser := setupTestApp(t)
	mockUow.On("Begin").Return(nil)
	accountRepo.On("Get", mock.Anything).Return(&domain.Account{}, domain.ErrAccountNotFound)

	// Test the route
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/deposit", uuid.New()), bytes.NewBuffer([]byte(`{"amount": 100.0}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	assert.Equal(fiber.StatusNotFound, resp.StatusCode)
	// mockUow.AssertCalled(t, "Rollback")
	mockUow.AssertExpectations(t)
	userRepo.AssertExpectations(t)
	accountRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)
}
func TestAccountRoutesFailureTransaction(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	app, userRepo, accountRepo, transactionRepo, mockUow, testUser := setupTestApp(t)
	mockUow.On("Begin").Return(nil)
	mockUow.On("Rollback").Return(nil)
	accountRepo.On("Get", mock.Anything).Return(&domain.Account{Balance: 100.0, UserID: testUser.ID}, nil)

	// test deposit negative amount
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/deposit", uuid.New()), bytes.NewBuffer([]byte(`{"amount": -100.0}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(err)
	defer resp.Body.Close() //nolint:errcheck
	assert.Equal(fiber.StatusBadRequest, resp.StatusCode)

	// test withdraw negative amount
	req = httptest.NewRequest("POST", fmt.Sprintf("/account/%s/withdraw", uuid.New()), bytes.NewBuffer([]byte(`{"amount": -100.0}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err = app.Test(req)
	require.NoError(err)
	defer resp.Body.Close() //nolint:errcheck
	assert.Equal(fiber.StatusBadRequest, resp.StatusCode)

	// test withdraw amount greater than balance
	req = httptest.NewRequest("POST", fmt.Sprintf("/account/%s/withdraw", uuid.New()), bytes.NewBuffer([]byte(`{"amount": 1000.0}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err = app.Test(req)
	require.NoError(err)
	defer resp.Body.Close() //nolint:errcheck
	assert.Equal(fiber.StatusUnprocessableEntity, resp.StatusCode, "Expected status %d, got %d", fiber.StatusUnprocessableEntity, resp.StatusCode)
	mockUow.AssertExpectations(t)
	userRepo.AssertExpectations(t)
	accountRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)
}

func TestAccountRoutesTransactionList(t *testing.T) {
	assert := assert.New(t)
	app, userRepo, accountRepo, transactionRepo, mockUow, testUser := setupTestApp(t)
	account := domain.NewAccount(testUser.ID)
	created1, _ := time.Parse(time.RFC3339, "2023-10-01T00:00:00Z")
	created2, _ := time.Parse(time.RFC3339, "2023-10-02T00:00:00Z")
	transactionRepo.On("List", testUser.ID, account.ID).Return([]*domain.Transaction{
		{ID: uuid.New(), Amount: 100.0, Balance: 100.0, Created: created1},
		{ID: uuid.New(), Amount: 50.0, Balance: 150.0, Created: created2},
	}, nil)

	req := httptest.NewRequest("GET", fmt.Sprintf("/account/%s/transactions", account.ID), nil)
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
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
	mockUow.AssertExpectations(t)
	userRepo.AssertExpectations(t)
	accountRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)
}

func TestAccountRoutesBalance(t *testing.T) {
	assert := assert.New(t)
	app, userRepo, accountRepo, transactionRepo, mockUow, testUser := setupTestApp(t)

	accountID := uuid.New()
	account := &domain.Account{ID: accountID, UserID: testUser.ID, Balance: 100.0}
	accountRepo.On("Get", accountID).Return(account, nil)
	req := httptest.NewRequest("GET", fmt.Sprintf("/account/%s/balance", accountID), nil)
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
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
	mockUow.AssertExpectations(t)
	userRepo.AssertExpectations(t)
	accountRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)

}

func TestAccountRoutesUoWError(t *testing.T) {
	app, userRepo, accountRepo, transactionRepo, mockUow, testUser := setupTestApp(t)
	mockUow.On("Begin").Return(errors.New("failed to begin transaction"))
	// Test the route
	req := httptest.NewRequest("POST", "/account", nil)
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", fiber.StatusInternalServerError, resp.StatusCode)
	}
	mockUow.AssertExpectations(t)
	userRepo.AssertExpectations(t)
	accountRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)
}
func TestGetBalanceAccountNotFound(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	app, userRepo, accountRepo, transactionRepo, mockUow, testUser := setupTestApp(t)
	accountRepo.On("Get", mock.Anything).Return(&domain.Account{}, domain.ErrAccountNotFound)
	// Test the route
	req := httptest.NewRequest("GET", fmt.Sprintf("/account/%s/balance", uuid.New()), nil)
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(err)
	defer resp.Body.Close() //nolint:errcheck
	assert.Equal(fiber.StatusNotFound, resp.StatusCode)
	mockUow.AssertExpectations(t)
	userRepo.AssertExpectations(t)
	accountRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)
}
func TestGetTransactions(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	app, userRepo, accountRepo, transactionRepo, mockUow, testUser := setupTestApp(t)
	transactionRepo.On("List", testUser.ID, mock.Anything).Return([]*domain.Transaction{}, nil)
	req := httptest.NewRequest("GET", fmt.Sprintf("/account/%s/transactions", uuid.New()), nil)
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(err)
	defer resp.Body.Close() //nolint:errcheck
	assert.Equal(fiber.StatusOK, resp.StatusCode)
	mockUow.AssertExpectations(t)
	userRepo.AssertExpectations(t)
	accountRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)

}

func TestAccountRoutesRollbackWhenCreateFails(t *testing.T) {
	app, userRepo, accountRepo, transactionRepo, mockUow, testUser := setupTestApp(t)
	mockUow.On("Begin").Return(nil)
	accountRepo.On("Create", mock.Anything).Return(errors.New("failed to create account"))
	// Test the route
	req := httptest.NewRequest("POST", "/account", nil)
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	mockUow.AssertExpectations(t)
	userRepo.AssertExpectations(t)
	accountRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)
}

func TestAccountRoutesRollbackWhenDepositFails(t *testing.T) {
	app, userRepo, accountRepo, transactionRepo, mockUow, testUser := setupTestApp(t)
	mockUow.On("Begin").Return(nil)
	account := domain.NewAccount(testUser.ID)
	accountRepo.On("Get", account.ID).Return(account, nil)
	transactionRepo.On("Create", mock.Anything).Return(errors.New("failed to create transaction"))
	accountRepo.On("Update", mock.Anything).Return(nil)
	// Test the route
	depositBody := bytes.NewBuffer([]byte(`{"amount": 100.0}`))
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/deposit", account.ID), depositBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	mockUow.AssertCalled(t, "Rollback")
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", fiber.StatusInternalServerError, resp.StatusCode)
	}
	mockUow.AssertExpectations(t)
	userRepo.AssertExpectations(t)
	accountRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)

}

func TestSimultaneousRequests(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	app, userRepo, accountRepo, transactionRepo, mockUow, testUser := setupTestApp(t)
	mockUow.On("Begin").Return(nil)
	mockUow.On("Commit").Return(nil)
	initialBalance := 1000.0
	acc := domain.NewAccount(testUser.ID)
	_, err := acc.Deposit(testUser.ID, initialBalance)
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
	token := getTestToken(t, app, userRepo, mockUow, testUser)

	for range numOperations {
		go func() {
			defer wg.Done()
			depositBody := bytes.NewBuffer(fmt.Appendf(nil, `{"amount": %f}`, depositAmount))
			req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/deposit", uuid.New()), depositBody)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			resp, depositErr := app.Test(req)
			if depositErr != nil {
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
			req.Header.Set("Authorization", "Bearer "+token)
			resp, withdrawErr := app.Test(req)
			if withdrawErr != nil {
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
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
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
	mockUow.AssertExpectations(t)
	userRepo.AssertExpectations(t)
	accountRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)
}

func getTestToken(t *testing.T, app *fiber.App, userRepo *UserMockRepo, mockUow *MockUoW, testUser *domain.User) string {
	t.Helper()
	setupCommonMocks(userRepo, mockUow, testUser)
	req := httptest.NewRequest("POST", "/login",
		bytes.NewBuffer([]byte(`{"identity":"testuser","password":"password123"}`)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var result struct {
		Token string `json:"token"`
	}
	fmt.Println("Response status:", resp.StatusCode)
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Response body:", string(bodyBytes))
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		t.Fatal(err)
	}
	token := result.Token
	return token
}
