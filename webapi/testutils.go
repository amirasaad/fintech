package webapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

func SetupTestApp(t *testing.T) (app *fiber.App, userRepo *fixtures.MockUserRepository, accountRepo *fixtures.MockAccountRepository, transactionRepo *fixtures.MockTransactionRepository, mockUow *fixtures.MockUnitOfWork, testUser *domain.User) {
	t.Helper()
	t.Setenv("JWT_SECRET_KEY", "secret")

	userRepo = fixtures.NewMockUserRepository(t)
	accountRepo = fixtures.NewMockAccountRepository(t)
	transactionRepo = fixtures.NewMockTransactionRepository(t)

	mockUow = fixtures.NewMockUnitOfWork(t)

	app = NewApp(func() (repository.UnitOfWork, error) { return mockUow, nil })
	testUser, _ = domain.NewUser("testuser", "testuser@example.com", "password123")
	log.SetOutput(io.Discard)
	defer mockUow.AssertExpectations(t)
	defer userRepo.AssertExpectations(t)
	defer accountRepo.AssertExpectations(t)
	defer transactionRepo.AssertExpectations(t)
	return
}

func getTestToken(t *testing.T, app *fiber.App, userRepo *fixtures.MockUserRepository, mockUow *fixtures.MockUnitOfWork, testUser *domain.User) string {
	t.Helper()
	mockUow.EXPECT().UserRepository().Return(userRepo)
	userRepo.EXPECT().GetByUsername("testuser").Return(testUser, nil)
	req := httptest.NewRequest("POST", "/login",
		bytes.NewBuffer([]byte(`{"identity":"testuser","password":"password123"}`)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	var result struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		t.Fatal(err)
	}
	token := result.Data.Token
	return token
}
