package webapi

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures"
	"github.com/stretchr/testify/suite"

	"os"
	"path/filepath"
	"runtime"

	fixturescurrency "github.com/amirasaad/fintech/internal/fixtures/currency"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

type E2ETestSuite struct {
	suite.Suite
}

func findNearestEnvTest() (current string, err error) {
	startDir, err := os.Getwd()
	if err != nil {
		return
	}
	curr := startDir
	for {
		candidate := filepath.Join(curr, ".env.test")
		if _, err = os.Stat(candidate); err == nil {
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

	testUser, err = user.NewUser("test", "test@example.com", "password123")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	logger := slog.Default()
	authStrategy := service.NewJWTAuthStrategy(mockUow, cfg.Jwt, logger)
	authService = service.NewAuthService(mockUow, authStrategy, logger)
	mockConverter = fixtures.NewMockCurrencyConverter(t)
	// Create services with the mock UOW factory
	accountSvc := service.NewAccountService(mockUow, mockConverter, logger)
	userSvc := service.NewUserService(mockUow, logger)

	// Initialize currency service with testing registry
	ctx := context.Background()
	var currencyRegistry *currency.CurrencyRegistry
	currencyRegistry, err = currency.NewCurrencyRegistry(ctx)
	if err != nil {
		t.Fatalf("Failed to create currency registry for tests: %v", err)
	}
	// Robustly resolve the fixture path
	_, filename, _, _ := runtime.Caller(0)
	fixturePath := filepath.Join(filepath.Dir(filename), "../internal/fixtures/currency/meta.csv")
	metas, err := fixturescurrency.LoadCurrencyMetaCSV(fixturePath)
	if err != nil {
		t.Fatalf("Failed to load currency meta fixture: %v", err)
	}
	for _, meta := range metas {
		if err := currencyRegistry.Register(meta); err != nil {
			t.Fatalf("Failed to register currency meta: %v", err)
		}
	}
	currencySvc := service.NewCurrencyService(currencyRegistry, logger)

	app = NewApp(accountSvc, userSvc, authService, currencySvc, cfg)
	log.SetOutput(io.Discard)

	return
}
