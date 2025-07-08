package webapi

import (
	"io"
	"log/slog"
	"runtime"
	"strings"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"os"
	"path/filepath"

	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
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

func findNearestEnvTest() (current string, err error) {
	startDir, err := os.Getwd()
	if err != nil {
		return
	}
	curr := startDir
	for {
		candidate := filepath.Join(curr, ".env.test")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	err = os.ErrNotExist
	return
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
	mockAuthStrategy *fixtures.MockAuthStrategy,
	mockConverter *fixtures.MockCurrencyConverter,
	cfg *config.AppConfig,
) {
	t.Helper()
	// Search for nearest .env.test upwards from current directory
	cfgPath, err := findNearestEnvTest()
	if err != nil {
		t.Fatalf("Failed to find .env.test for tests: %v", err)
	}
	cfg, err = config.LoadAppConfig(slog.Default(), cfgPath)
	if err != nil {
		t.Fatalf("Failed to load app config for tests: %v", err)
	}

	userRepo = fixtures.NewMockUserRepository(t)
	accountRepo = fixtures.NewMockAccountRepository(t)
	transactionRepo = fixtures.NewMockTransactionRepository(t)

	mockUow = fixtures.NewMockUnitOfWork(t)

	testUser, err = domain.NewUser("test", "test@example.com", "password123")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	uow := func() (repository.UnitOfWork, error) { return mockUow, nil }
	authStrategy := service.NewJWTAuthStrategy(uow, cfg.Jwt)
	authService = service.NewAuthService(uow, authStrategy)
	mockConverter = fixtures.NewMockCurrencyConverter(t)
	// Create services with the mock UOW factory
	accountSvc := service.NewAccountService(uow, mockConverter)
	userSvc := service.NewUserService(uow)

	app = NewApp(accountSvc, userSvc, authService, cfg)
	log.SetOutput(io.Discard)

	return
}
