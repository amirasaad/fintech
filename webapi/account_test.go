package webapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/gofiber/fiber/v2"
)

func TestAccountCreate(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	app, userRepo, accountRepo, _, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().AccountRepository().Return(accountRepo)
	accountRepo.On("Create", mock.Anything).Return(nil)

	mockUow.On("Begin").Return(nil)
	mockUow.On("Commit").Return(nil)

	req := httptest.NewRequest("POST", "/account", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))

	resp, err := app.Test(req)
	require.NoError(err)
	defer resp.Body.Close() //nolint:errcheck
	assert.Equal(fiber.StatusCreated, resp.StatusCode)

}

func TestAccountDeposit(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	app, userRepo, accountRepo, transactionRepo, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().AccountRepository().Return(accountRepo)
	mockUow.EXPECT().TransactionRepository().Return(transactionRepo)

	// Always return an account with the correct ID and UserID
	testAccount := domain.NewAccount(testUser.ID)
	accountRepo.On("Get", mock.Anything).Return(testAccount, nil)
	transactionRepo.On("Create", mock.Anything).Return(nil)
	accountRepo.On("Update", mock.Anything).Return(nil)
	mockUow.On("Begin").Return(nil)
	mockUow.On("Commit").Return(nil)

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

	app, userRepo, accountRepo, transactionRepo, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().AccountRepository().Return(accountRepo)
	mockUow.EXPECT().TransactionRepository().Return(transactionRepo)

	testAccount := domain.NewAccount(testUser.ID)
	_, _ = testAccount.Deposit(testUser.ID, 1000)
	accountRepo.On("Get", mock.Anything).Return(testAccount, nil)
	transactionRepo.On("Create", mock.Anything).Return(nil)
	accountRepo.On("Update", mock.Anything).Return(nil)
	mockUow.On("Begin").Return(nil)
	mockUow.On("Commit").Return(nil)

	withdrawBody := bytes.NewBuffer([]byte(`{"amount": 100.0}`))
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/withdraw", testAccount.ID), withdrawBody)
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
	app, userRepo, accountRepo, transactionRepo, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().AccountRepository().Return(accountRepo)
	mockUow.On("Begin").Return(nil)
	mockUow.EXPECT().Rollback().Return(nil).Once()
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
	app, userRepo, accountRepo, transactionRepo, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().AccountRepository().Return(accountRepo)
	mockUow.On("Begin").Return(nil)
	mockUow.On("Rollback").Return(nil)
	accountRepo.On("Get", mock.Anything).Return(&domain.Account{Balance: 100.0, UserID: testUser.ID}, nil)

	// fixtures deposit negative amount
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/deposit", uuid.New()), bytes.NewBuffer([]byte(`{"amount": -100.0}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(err)
	defer resp.Body.Close() //nolint:errcheck
	assert.Equal(fiber.StatusBadRequest, resp.StatusCode)

	// fixtures withdraw negative amount
	req = httptest.NewRequest("POST", fmt.Sprintf("/account/%s/withdraw", uuid.New()), bytes.NewBuffer([]byte(`{"amount": -100.0}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err = app.Test(req)
	require.NoError(err)
	defer resp.Body.Close() //nolint:errcheck
	assert.Equal(fiber.StatusBadRequest, resp.StatusCode)

	mockUow.AssertExpectations(t)
	userRepo.AssertExpectations(t)
	accountRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)
}

func TestAccountRoutesTransactionList(t *testing.T) {
	assert := assert.New(t)
	app, userRepo, accountRepo, transactionRepo, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().TransactionRepository().Return(transactionRepo)
	account := domain.NewAccount(testUser.ID)
	created1, _ := time.Parse(time.RFC3339, "2023-10-01T00:00:00Z")
	created2, _ := time.Parse(time.RFC3339, "2023-10-02T00:00:00Z")
	transactionRepo.On("List", testUser.ID, account.ID).Return([]*domain.Transaction{
		{ID: uuid.New(), Amount: 100.0, Balance: 100.0, CreatedAt: created1},
		{ID: uuid.New(), Amount: 50.0, Balance: 150.0, CreatedAt: created2},
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

	var response Response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	assert.NotNil(response.Data, "Should contain list of transactions")
	mockUow.AssertExpectations(t)
	userRepo.AssertExpectations(t)
	accountRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)
}

func TestAccountRoutesBalance(t *testing.T) {
	assert := assert.New(t)
	app, userRepo, accountRepo, transactionRepo, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().AccountRepository().Return(accountRepo)
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
	var response Response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	balanceMap, ok := response.Data.(map[string]interface{})
	assert.True(ok, "Expected response.Data to be a map")
	balance, ok := balanceMap["balance"].(float64)
	assert.True(ok, "Expected balance to be a float64")

	assert.Equal(float64(account.Balance/100.0), balance, "Expected balance to match")
	mockUow.AssertExpectations(t)
	userRepo.AssertExpectations(t)
	accountRepo.AssertExpectations(t)
	transactionRepo.AssertExpectations(t)

}

func TestAccountRoutesUoWError(t *testing.T) {
	app, userRepo, _, _, mockUow, testUser := SetupTestApp(t)
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

}
func TestGetBalanceAccountNotFound(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	app, userRepo, accountRepo, _, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().AccountRepository().Return(accountRepo)
	accountRepo.EXPECT().Get(mock.Anything).Return(&domain.Account{}, domain.ErrAccountNotFound)
	// Test the route
	req := httptest.NewRequest("GET", fmt.Sprintf("/account/%s/balance", uuid.New()), nil)
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(err)
	defer resp.Body.Close() //nolint:errcheck
	assert.Equal(fiber.StatusNotFound, resp.StatusCode)
}
func TestGetTransactions(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	app, userRepo, _, transactionRepo, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().TransactionRepository().Return(transactionRepo)
	transactionRepo.On("List", testUser.ID, mock.Anything).Return([]*domain.Transaction{}, nil)
	req := httptest.NewRequest("GET", fmt.Sprintf("/account/%s/transactions", uuid.New()), nil)
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(err)
	defer resp.Body.Close() //nolint:errcheck
	assert.Equal(fiber.StatusOK, resp.StatusCode)

}

func TestAccountRoutesRollbackWhenCreateFails(t *testing.T) {
	assert := assert.New(t)
	app, userRepo, accountRepo, _, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().AccountRepository().Return(accountRepo)
	mockUow.On("Begin").Return(nil)
	mockUow.On("Rollback").Return(nil)
	accountRepo.On("Create", mock.Anything).Return(errors.New("failed to create account"))
	// Test the route
	req := httptest.NewRequest("POST", "/account", nil)
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	assert.Equal(fiber.StatusInternalServerError, resp.StatusCode)

}

func TestAccountRoutesRollbackWhenDepositFails(t *testing.T) {
	app, userRepo, accountRepo, transactionRepo, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().AccountRepository().Return(accountRepo)
	mockUow.EXPECT().TransactionRepository().Return(transactionRepo)
	mockUow.EXPECT().Begin().Return(nil)
	mockUow.EXPECT().Rollback().Return(nil)
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

}

func TestCreateAccount_Unauthorized(t *testing.T) {
	app, _, _, _, _, _ := SetupTestApp(t)
	req := httptest.NewRequest("POST", "/account", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestCreateAccount_InvalidUserID(t *testing.T) {
	app, userRepo, _, _, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().UserRepository().Return(userRepo).Once()
	userRepo.EXPECT().GetByUsername("testuser").Return(testUser, nil).Once()
	mockUow.EXPECT().Begin().Return(errors.New("uow error")).Once()

	req := httptest.NewRequest("POST", "/account", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestDeposit_Unauthorized(t *testing.T) {
	app, _, _, _, _, _ := SetupTestApp(t)
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/deposit", uuid.New()), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestDeposit_InvalidAccountID(t *testing.T) {
	app, userRepo, _, _, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().UserRepository().Return(userRepo).Once()
	userRepo.EXPECT().GetByUsername("testuser").Return(testUser, nil).Once()

	depositBody := bytes.NewBuffer([]byte(`{"amount": 100.0}`))
	req := httptest.NewRequest("POST", "/account/invalid-uuid/deposit", depositBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestDeposit_InvalidBody(t *testing.T) {
	app, userRepo, _, _, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().UserRepository().Return(userRepo).Once()
	userRepo.EXPECT().GetByUsername("testuser").Return(testUser, nil).Once()

	depositBody := bytes.NewBuffer([]byte(`{"amount":"invalid"}`)) // Invalid body
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/deposit", uuid.New()), depositBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestWithdraw_Unauthorized(t *testing.T) {
	app, _, _, _, _, _ := SetupTestApp(t)
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/withdraw", uuid.New()), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestWithdraw_InvalidAccountID(t *testing.T) {
	app, userRepo, _, _, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().UserRepository().Return(userRepo).Once()
	userRepo.EXPECT().GetByUsername("testuser").Return(testUser, nil).Once()

	withdrawBody := bytes.NewBuffer([]byte(`{"amount": 100.0}`))
	req := httptest.NewRequest("POST", "/account/invalid-uuid/withdraw", withdrawBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestWithdraw_InvalidBody(t *testing.T) {
	app, userRepo, _, _, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().UserRepository().Return(userRepo).Once()
	userRepo.EXPECT().GetByUsername("testuser").Return(testUser, nil).Once()

	withdrawBody := bytes.NewBuffer([]byte(`{"amount":"invalid"}`)) // Invalid body
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/withdraw", uuid.New()), withdrawBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetTransactions_Unauthorized(t *testing.T) {
	app, _, _, _, _, _ := SetupTestApp(t)
	req := httptest.NewRequest("GET", fmt.Sprintf("/account/%s/transactions", uuid.New()), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestGetTransactions_InvalidAccountID(t *testing.T) {
	app, userRepo, _, _, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().UserRepository().Return(userRepo).Once()
	userRepo.EXPECT().GetByUsername("testuser").Return(testUser, nil).Once()

	req := httptest.NewRequest("GET", "/account/invalid-uuid/transactions", nil)
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetTransactions_InternalServerError(t *testing.T) {
	app, userRepo, _, transactionRepo, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().UserRepository().Return(userRepo).Once()
	userRepo.EXPECT().GetByUsername("testuser").Return(testUser, nil).Once()
	mockUow.EXPECT().TransactionRepository().Return(transactionRepo).Once()
	transactionRepo.On("List", mock.Anything, mock.Anything).Return(nil, errors.New("db error")).Once()

	req := httptest.NewRequest("GET", fmt.Sprintf("/account/%s/transactions", uuid.New()), nil)
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestGetBalance_Unauthorized(t *testing.T) {
	app, _, _, _, _, _ := SetupTestApp(t)
	req := httptest.NewRequest("GET", fmt.Sprintf("/account/%s/balance", uuid.New()), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestGetBalance_InvalidAccountID(t *testing.T) {
	app, userRepo, _, _, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().UserRepository().Return(userRepo).Once()
	userRepo.EXPECT().GetByUsername("testuser").Return(testUser, nil).Once()

	req := httptest.NewRequest("GET", "/account/invalid-uuid/balance", nil)
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetBalance_InternalServerError(t *testing.T) {
	app, userRepo, accountRepo, _, mockUow, testUser := SetupTestApp(t)
	mockUow.EXPECT().UserRepository().Return(userRepo).Once()
	userRepo.EXPECT().GetByUsername("testuser").Return(testUser, nil).Once()
	mockUow.EXPECT().AccountRepository().Return(accountRepo).Once()
	accountRepo.On("Get", mock.Anything).Return(nil, errors.New("db error")).Once()

	req := httptest.NewRequest("GET", fmt.Sprintf("/account/%s/balance", uuid.New()), nil)
	req.Header.Set("Authorization", "Bearer "+getTestToken(t, app, userRepo, mockUow, testUser))
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}
