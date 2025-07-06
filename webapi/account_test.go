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

	"github.com/amirasaad/fintech/internal/fixtures"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type AccountTestSuite struct {
	E2ETestSuite
	app         *fiber.App
	userRepo    *fixtures.MockUserRepository
	accountRepo *fixtures.MockAccountRepository
	transRepo   *fixtures.MockTransactionRepository
	mockUow     *fixtures.MockUnitOfWork
	testUser    *domain.User
	testToken   string
}

func (s *AccountTestSuite) BeforeTest(_, testName string) {
	s.E2ETestSuite.BeforeTest("", testName)
	s.app, s.userRepo, s.accountRepo, s.transRepo, s.mockUow, s.testUser = SetupTestApp(s.T())
	s.testToken = getTestToken(s.T(), s.app, s.userRepo, s.mockUow, s.testUser)

}

func (s *AccountTestSuite) TestAccountCreate() {
	s.mockUow.EXPECT().AccountRepository().Return(s.accountRepo)
	s.accountRepo.On("Create", mock.Anything).Return(nil)

	s.mockUow.On("Begin").Return(nil)
	s.mockUow.On("Commit").Return(nil)

	req := httptest.NewRequest("POST", "/account", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close() //nolint: errcheck

	s.Assert().Equal(fiber.StatusCreated, resp.StatusCode)
}

func (s *AccountTestSuite) TestAccountDeposit() {
	s.mockUow.EXPECT().AccountRepository().Return(s.accountRepo)
	s.mockUow.EXPECT().TransactionRepository().Return(s.transRepo)

	testAccount := domain.NewAccount(s.testUser.ID)
	s.accountRepo.On("Get", mock.Anything).Return(testAccount, nil)
	s.transRepo.On("Create", mock.Anything).Return(nil)
	s.accountRepo.On("Update", mock.Anything).Return(nil)
	s.mockUow.On("Begin").Return(nil)
	s.mockUow.On("Commit").Return(nil)

	depositBody := bytes.NewBuffer([]byte(`{"amount": 100.0}`))
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/deposit", testAccount.ID), depositBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(fiber.StatusOK, resp.StatusCode)
}

func (s *AccountTestSuite) TestAccountWithdraw() {
	s.mockUow.EXPECT().AccountRepository().Return(s.accountRepo)
	s.mockUow.EXPECT().TransactionRepository().Return(s.transRepo)

	testAccount := domain.NewAccount(s.testUser.ID)
	_, _ = testAccount.Deposit(s.testUser.ID, 1000)
	s.accountRepo.On("Get", mock.Anything).Return(testAccount, nil)
	s.transRepo.On("Create", mock.Anything).Return(nil)
	s.accountRepo.On("Update", mock.Anything).Return(nil)
	s.mockUow.On("Begin").Return(nil)
	s.mockUow.On("Commit").Return(nil)

	withdrawBody := bytes.NewBuffer([]byte(`{"amount": 100.0}`))
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/withdraw", testAccount.ID), withdrawBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(fiber.StatusOK, resp.StatusCode)
}

func (s *AccountTestSuite) TestAccountRoutesFailureAccountNotFound() {
	s.mockUow.EXPECT().AccountRepository().Return(s.accountRepo)
	s.mockUow.On("Begin").Return(nil)
	s.mockUow.EXPECT().Rollback().Return(nil).Once()
	s.accountRepo.On("Get", mock.Anything).Return(&domain.Account{}, domain.ErrAccountNotFound)

	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/deposit", uuid.New()), bytes.NewBuffer([]byte(`{"amount": 100.0}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(fiber.StatusNotFound, resp.StatusCode)
	s.mockUow.AssertCalled(s.T(), "Rollback")
}

func (s *AccountTestSuite) TestAccountRoutesFailureTransaction() {
	s.mockUow.EXPECT().AccountRepository().Return(s.accountRepo)
	s.mockUow.On("Begin").Return(nil)
	s.mockUow.On("Rollback").Return(nil)
	s.accountRepo.On("Get", mock.Anything).Return(&domain.Account{Balance: 100.0, UserID: s.testUser.ID}, nil)

	// fixtures deposit negative amount
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/deposit", uuid.New()), bytes.NewBuffer([]byte(`{"amount": -100.0}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(fiber.StatusBadRequest, resp.StatusCode)

	// fixtures withdraw negative amount
	req = httptest.NewRequest("POST", fmt.Sprintf("/account/%s/withdraw", uuid.New()), bytes.NewBuffer([]byte(`{"amount": -100.0}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err = s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(fiber.StatusBadRequest, resp.StatusCode)
}

func (s *AccountTestSuite) TestAccountRoutesTransactionList() {
	s.mockUow.EXPECT().TransactionRepository().Return(s.transRepo)
	account := domain.NewAccount(s.testUser.ID)
	created1, _ := time.Parse(time.RFC3339, "2023-10-01T00:00:00Z")
	created2, _ := time.Parse(time.RFC3339, "2023-10-02T00:00:00Z")
	s.transRepo.On("List", s.testUser.ID, account.ID).Return([]*domain.Transaction{
		{ID: uuid.New(), Amount: 100.0, Balance: 100.0, CreatedAt: created1},
		{ID: uuid.New(), Amount: 50.0, Balance: 150.0, CreatedAt: created2},
	}, nil)

	req := httptest.NewRequest("GET", fmt.Sprintf("/account/%s/transactions", account.ID), nil)
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(fiber.StatusOK, resp.StatusCode)

	var response Response
	bodyBytes, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	err = json.Unmarshal(bodyBytes, &response)
	s.Require().NoError(err)

	s.Assert().NotNil(response.Data, "Should contain list of transactions")
}

func (s *AccountTestSuite) TestAccountRoutesBalance() {
	s.mockUow.EXPECT().AccountRepository().Return(s.accountRepo)
	accountID := uuid.New()
	account := &domain.Account{ID: accountID, UserID: s.testUser.ID, Balance: 100.0}
	s.accountRepo.On("Get", accountID).Return(account, nil)

	req := httptest.NewRequest("GET", fmt.Sprintf("/account/%s/balance", accountID), nil)
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(fiber.StatusOK, resp.StatusCode)

	var response Response
	bodyBytes, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	err = json.Unmarshal(bodyBytes, &response)
	s.Require().NoError(err)

	balanceMap, ok := response.Data.(map[string]any)
	s.Assert().True(ok, "Expected response.Data to be a map")

	balance, ok := balanceMap["balance"].(float64)
	s.Assert().True(ok, "Expected balance to be a float64")

	s.Assert().Equal(float64(account.Balance/100.0), balance, "Expected balance to match")
}

func (s *AccountTestSuite) TestAccountRoutesUoWError() {
	s.mockUow.On("Begin").Return(errors.New("failed to begin transaction"))

	req := httptest.NewRequest("POST", "/account", nil)
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(fiber.StatusInternalServerError, resp.StatusCode)
}

func (s *AccountTestSuite) TestGetBalanceAccountNotFound() {
	s.mockUow.EXPECT().AccountRepository().Return(s.accountRepo)
	s.accountRepo.EXPECT().Get(mock.Anything).Return(&domain.Account{}, domain.ErrAccountNotFound)

	req := httptest.NewRequest("GET", fmt.Sprintf("/account/%s/balance", uuid.New()), nil)
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(fiber.StatusNotFound, resp.StatusCode)
}

func (s *AccountTestSuite) TestGetTransactions() {
	s.mockUow.EXPECT().TransactionRepository().Return(s.transRepo)
	s.transRepo.On("List", s.testUser.ID, mock.Anything).Return([]*domain.Transaction{}, nil)

	req := httptest.NewRequest("GET", fmt.Sprintf("/account/%s/transactions", uuid.New()), nil)
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(fiber.StatusOK, resp.StatusCode)
}

func (s *AccountTestSuite) TestAccountRoutesRollbackWhenCreateFails() {
	s.mockUow.EXPECT().AccountRepository().Return(s.accountRepo)
	s.mockUow.On("Begin").Return(nil)
	s.mockUow.On("Rollback").Return(nil)
	s.accountRepo.On("Create", mock.Anything).Return(errors.New("failed to create account"))

	req := httptest.NewRequest("POST", "/account", nil)
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(fiber.StatusInternalServerError, resp.StatusCode)
}

func (s *AccountTestSuite) TestAccountRoutesRollbackWhenDepositFails() {
	s.mockUow.EXPECT().AccountRepository().Return(s.accountRepo)
	s.mockUow.EXPECT().TransactionRepository().Return(s.transRepo)
	s.mockUow.EXPECT().Begin().Return(nil)
	s.mockUow.EXPECT().Rollback().Return(nil)
	account := domain.NewAccount(s.testUser.ID)
	s.accountRepo.On("Get", account.ID).Return(account, nil)
	s.transRepo.On("Create", mock.Anything).Return(errors.New("failed to create transaction"))
	s.accountRepo.On("Update", mock.Anything).Return(nil)

	depositBody := bytes.NewBuffer([]byte(`{"amount": 100.0}`))
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/deposit", account.ID), depositBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.mockUow.AssertCalled(s.T(), "Rollback")
	s.Assert().Equal(fiber.StatusInternalServerError, resp.StatusCode)
}

func (s *AccountTestSuite) TestCreateAccount_Unauthorized() {
	req := httptest.NewRequest("POST", "/account", nil)
	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	s.Assert().Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *AccountTestSuite) TestCreateAccount_InvalidUserID() {
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo)
	s.userRepo.EXPECT().GetByUsername("testuser").Return(s.testUser, nil)
	s.mockUow.EXPECT().Begin().Return(errors.New("uow error"))

	req := httptest.NewRequest("POST", "/account", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)
	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	s.Assert().Equal(fiber.StatusInternalServerError, resp.StatusCode)
}

func (s *AccountTestSuite) TestDeposit_Unauthorized() {

	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/deposit", uuid.New()), nil)
	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	s.Assert().Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *AccountTestSuite) TestDeposit_InvalidAccountID() {
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo)
	s.userRepo.EXPECT().GetByUsername("testuser").Return(s.testUser, nil)

	depositBody := bytes.NewBuffer([]byte(`{"amount": 100.0}`))
	req := httptest.NewRequest("POST", "/account/invalid-uuid/deposit", depositBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	s.Assert().Equal(fiber.StatusBadRequest, resp.StatusCode)
}

func (s *AccountTestSuite) TestDeposit_InvalidBody() {
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo)
	s.userRepo.EXPECT().GetByUsername("testuser").Return(s.testUser, nil)

	depositBody := bytes.NewBuffer([]byte(`{"amount":"invalid"}`))
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/deposit", uuid.New()), depositBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	s.Assert().Equal(fiber.StatusBadRequest, resp.StatusCode)
}

func (s *AccountTestSuite) TestWithdraw_Unauthorized() {
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/withdraw", uuid.New()), nil)
	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	s.Assert().Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *AccountTestSuite) TestWithdraw_InvalidAccountID() {
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo)
	s.userRepo.EXPECT().GetByUsername("testuser").Return(s.testUser, nil)

	withdrawBody := bytes.NewBuffer([]byte(`{"amount": 100.0}`))
	req := httptest.NewRequest("POST", "/account/invalid-uuid/withdraw", withdrawBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	s.Assert().Equal(fiber.StatusBadRequest, resp.StatusCode)
}

func (s *AccountTestSuite) TestWithdraw_InvalidBody() {
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo)
	s.userRepo.EXPECT().GetByUsername("testuser").Return(s.testUser, nil)

	withdrawBody := bytes.NewBuffer([]byte(`{"amount":"invalid"}`))
	req := httptest.NewRequest("POST", fmt.Sprintf("/account/%s/withdraw", uuid.New()), withdrawBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	s.Assert().Equal(fiber.StatusBadRequest, resp.StatusCode)
}

func (s *AccountTestSuite) TestGetTransactions_Unauthorized() {
	req := httptest.NewRequest("GET", fmt.Sprintf("/account/%s/transactions", uuid.New()), nil)
	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	s.Assert().Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *AccountTestSuite) TestGetTransactions_InvalidAccountID() {
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo)
	s.userRepo.EXPECT().GetByUsername("testuser").Return(s.testUser, nil)

	req := httptest.NewRequest("GET", "/account/invalid-uuid/transactions", nil)
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	s.Assert().Equal(fiber.StatusBadRequest, resp.StatusCode)
}

func (s *AccountTestSuite) TestGetTransactions_InternalServerError() {
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo)
	s.userRepo.EXPECT().GetByUsername("testuser").Return(s.testUser, nil)
	s.mockUow.EXPECT().TransactionRepository().Return(s.transRepo)
	s.transRepo.On("List", mock.Anything, mock.Anything).Return(nil, errors.New("db error")).Once()

	req := httptest.NewRequest("GET", fmt.Sprintf("/account/%s/transactions", uuid.New()), nil)
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	s.Assert().Equal(fiber.StatusInternalServerError, resp.StatusCode)
}

func (s *AccountTestSuite) TestGetBalance_Unauthorized() {
	req := httptest.NewRequest("GET", fmt.Sprintf("/account/%s/balance", uuid.New()), nil)
	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	s.Assert().Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *AccountTestSuite) TestGetBalance_InvalidAccountID() {
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo)
	s.userRepo.EXPECT().GetByUsername("testuser").Return(s.testUser, nil)

	req := httptest.NewRequest("GET", "/account/invalid-uuid/balance", nil)
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	s.Assert().Equal(fiber.StatusBadRequest, resp.StatusCode)
}

func (s *AccountTestSuite) TestGetBalance_InternalServerError() {
	s.mockUow.EXPECT().UserRepository().Return(s.userRepo)
	s.userRepo.EXPECT().GetByUsername("testuser").Return(s.testUser, nil)
	s.mockUow.EXPECT().AccountRepository().Return(s.accountRepo)
	s.accountRepo.On("Get", mock.Anything).Return(nil, errors.New("db error"))

	req := httptest.NewRequest("GET", fmt.Sprintf("/account/%s/balance", uuid.New()), nil)
	req.Header.Set("Authorization", "Bearer "+s.testToken)

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)
	s.Assert().Equal(fiber.StatusInternalServerError, resp.StatusCode)
}

func TestAccountTestSuite(t *testing.T) {
	suite.Run(t, new(AccountTestSuite))
}
