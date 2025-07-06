package webapi

import (
	"io"
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
func NewTestApp(uowFactory func() (repository.UnitOfWork, error), strategy service.AuthStrategy) *fiber.App {
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

	AccountRoutes(app, uowFactory, strategy)
	UserRoutes(app, uowFactory, strategy)
	AuthRoutes(app, uowFactory, strategy)

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
	userRepo = fixtures.NewMockUserRepository(t)
	accountRepo = fixtures.NewMockAccountRepository(t)
	transactionRepo = fixtures.NewMockTransactionRepository(t)

	mockUow = fixtures.NewMockUnitOfWork(t)

	authStrategy := service.NewJWTAuthStrategy(func() (repository.UnitOfWork, error) { return mockUow, nil })
	authService = service.NewAuthService(func() (repository.UnitOfWork, error) { return mockUow, nil }, authStrategy)

	app = NewTestApp(func() (repository.UnitOfWork, error) { return mockUow, nil }, authStrategy)
	testUser, _ = domain.NewUser("testuser", "testuser@example.com", "password123")
	log.SetOutput(io.Discard)
	// os.Setenv("JWT_SECRET_KEY", "secret")

	return
}

func getTestToken(t *testing.T, authService *service.AuthService, userRepo *fixtures.MockUserRepository, mockUow *fixtures.MockUnitOfWork, testUser *domain.User) string {
	t.Helper()
	mockUow.EXPECT().UserRepository().Return(userRepo).Maybe()
	userRepo.EXPECT().GetByUsername("testuser").Return(testUser, nil).Maybe()

	_, token, err := authService.Login("testuser", "password123")
	if err != nil {
		t.Fatal(err)
	}
	return token
}
