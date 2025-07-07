package webapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

type E2ETestSuite struct {
	suite.Suite
	ts map[string]*testing.T // Map of test names > *testing.T
}

func (suite *E2ETestSuite) BeforeTest(_, testName string) {
	t := suite.T()
	if suite.ts == nil {
		suite.ts = make(map[string]*testing.T, 1)
	}
	suite.ts[testName] = t
	suite.T().Cleanup(func() {
		mock.AssertExpectationsForObjects(suite.T())
	})
	// Removed t.Parallel() to avoid concurrency issues with mocks
}

// T() overrides suite.Suite.T() with a way to find the proper *testing.T
// for the current test.
// This relies on `BeforeTest` storing the *testing.T pointers in a map
// before marking them parallel.
// This is a huge hack to make parallel testing work until
// https://github.com/stretchr/testify/issues/187 is fixed.
// There is still a small race:
// 1. test 1 calls SetT()
// 2. test 1 calls BeforeTest() with its own T
// 3. test 1 is marked as parallel and starts executing
// 4. test 2 calls SetT()
// 5. test 1 completes and calls SetT() to reset to the parent T
// 6. test 2 calls BeforeTest() with its parent T instead of its own
// The time between 4. & 6. is extremely low, enough that this should be really rare on our e2e tests.
func (suite *E2ETestSuite) T() *testing.T {
	// Try to find in the call stack a method name that is stored in `ts` (the test method).
	for i := 1; ; i++ {
		pc, _, _, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// Example rawFuncName:
		// github.com/foo/bar/tests/e2e.(*E2ETestSuite).MyTest
		rawFuncName := runtime.FuncForPC(pc).Name()
		splittedFuncName := strings.Split(rawFuncName, ".")
		funcName := splittedFuncName[len(splittedFuncName)-1]
		t := suite.ts[funcName]
		if t != nil {
			return t
		}
	}
	// Fallback to the globally stored Suite.T()
	return suite.Suite.T()
}

// NewTestApp creates a new Fiber app for testing without rate limiting
func NewTestApp(
	accountSvc *service.AccountService,
	userSvc *service.UserService,
	authSvc *service.AuthService,
) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// Default to 500 if status code cannot be determined
			status := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				status = e.Code
			}
			return ErrorResponseJSON(c, status, "Internal Server Error", err.Error())
		},
	})
	// No rate limiting for tests
	app.Use(recover.New())

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("App is working! 🚀")
	})

	AccountRoutes(app, accountSvc, authSvc)
	UserRoutes(app, userSvc, authSvc)
	AuthRoutes(app, authSvc)

	return app
}

func SetupTestApp(
	t *testing.T,
) (
	app *fiber.App,
	userRepo *fixtures.MockUserRepository,
	accountRepo *fixtures.MockAccountRepository,
	transactionRepo *fixtures.MockTransactionRepository,
	mockUow *fixtures.MockUnitOfWork,
	testUser *domain.User,
	authService *service.AuthService,
) {
	t.Helper()
	// Set JWT secret key before creating the app
	t.Setenv("JWT_SECRET_KEY", "secret")
	
	userRepo = fixtures.NewMockUserRepository(t)
	accountRepo = fixtures.NewMockAccountRepository(t)
	transactionRepo = fixtures.NewMockTransactionRepository(t)

	mockUow = fixtures.NewMockUnitOfWork(t)

	authStrategy := service.NewJWTAuthStrategy(func() (repository.UnitOfWork, error) { return mockUow, nil })
	authService = service.NewAuthService(func() (repository.UnitOfWork, error) { return mockUow, nil }, authStrategy)

	// Create services with the mock UOW factory
	accountSvc := service.NewAccountService(func() (repository.UnitOfWork, error) { return mockUow, nil })
	userSvc := service.NewUserService(func() (repository.UnitOfWork, error) { return mockUow, nil })

	app = NewTestApp(accountSvc, userSvc, authService)
	testUser, _ = domain.NewUser("testuser", "testuser@example.com", "password123")
	log.SetOutput(io.Discard)

	return
}

func getTestToken(
	t *testing.T,
	app *fiber.App,
	testUser *domain.User,
) string {
	t.Helper()
	loginBody := &LoginInput{Identity: testUser.Username, Password: "password123"}
	body, _ := json.Marshal(loginBody)

	req := httptest.NewRequest("POST", "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, 10000)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		// Read response body for debugging
		respBody, _ := io.ReadAll(resp.Body)
		t.Logf("Login failed with status %d, body: %s", resp.StatusCode, string(respBody))
		t.Fatalf("expected status 200 but got %d", resp.StatusCode)
	}
	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	token, ok := response.Data.(map[string]interface{})["token"].(string)
	if !ok {
		t.Fatal("unable to extract token from response")
	}
	return token
}
