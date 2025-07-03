package webapi

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/test"
	"github.com/gofiber/fiber/v2"
)

func SetupCommonMocks(userRepo *test.MockUserRepository, mockUow *test.MockUnitOfWork, testUser *domain.User) {
	// placeholder for common mocks
}

func SetupTestApp(t *testing.T) (app *fiber.App, userRepo *test.MockUserRepository, accountRepo *test.MockAccountRepository, transactionRepo *test.MockTransactionRepository, mockUow *test.MockUnitOfWork, testUser *domain.User) {
	t.Helper()
	t.Setenv("JWT_SECRET_KEY", "secret")

	userRepo = test.NewMockUserRepository(t)
	accountRepo = test.NewMockAccountRepository(t)
	transactionRepo = test.NewMockTransactionRepository(t)

	mockUow = test.NewMockUnitOfWork(t)

	app = NewApp(func() (repository.UnitOfWork, error) { return mockUow, nil })
	testUser, _ = domain.NewUser("testuser", "testuser@example.com", "password123")

	defer mockUow.AssertExpectations(t)
	defer userRepo.AssertExpectations(t)
	defer accountRepo.AssertExpectations(t)
	defer transactionRepo.AssertExpectations(t)
	return
}
