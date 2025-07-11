package webapi

import (
	"context"
	"io"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"os"
	"path/filepath"
	"runtime"

	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/amirasaad/fintech/infra/provider"
	infrarepo "github.com/amirasaad/fintech/infra/repository"
	fixturescurrency "github.com/amirasaad/fintech/internal/fixtures/currency"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/user"
	pkgrepo "github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// E2ETestSuite provides a base test suite for E2E tests
type E2ETestSuite struct {
	suite.Suite
}

// E2ETestSuiteWithDB provides a test suite with a real Postgres database using Testcontainers
type E2ETestSuiteWithDB struct {
	suite.Suite
	pgContainer *tcpostgres.PostgresContainer
	db          *gorm.DB
	app         *fiber.App
	testUser    *domain.User
	authService *service.AuthService
	cfg         *config.AppConfig
}

// generateRandomTestUser creates a test user with random username and email
func generateRandomTestUser() (*domain.User, error) {
	randomID := uuid.New().String()[:8]
	username := fmt.Sprintf("testuser_%s", randomID)
	email := fmt.Sprintf("test_%s@example.com", randomID)
	return user.NewUser(username, email, "password123")
}

// findNearestEnvTest searches for the nearest .env.test file in the directory tree
func findNearestEnvTest() (string, error) {
	startDir, err := os.Getwd()
	if err != nil {
		return "", err
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
	return "", os.ErrNotExist
}

// startPostgresContainer starts a Postgres container using Testcontainers
func startPostgresContainer(ctx context.Context) (*tcpostgres.PostgresContainer, error) {
	return tcpostgres.Run(
		ctx,
		"postgres:15-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(30*time.Second),
		),
	)
}

// runMigrations runs database migrations on the provided database
func runMigrations(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	driver, err := migratepostgres.WithInstance(sqlDB, &migratepostgres.Config{})
	if err != nil {
		return err
	}

	_, filename, _, _ := runtime.Caller(0)
	migrationsPath := filepath.Join(filepath.Dir(filename), "../internal/migrations")

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres", driver)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

// SetupTestAppWithTestcontainers creates a test app using real Postgres via Testcontainers
func SetupTestAppWithTestcontainers(t *testing.T) (*fiber.App, *gorm.DB, *domain.User, *service.AuthService, *config.AppConfig) {
	t.Helper()
	ctx := context.Background()

	// Start Postgres container
	pg, err := startPostgresContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start Postgres container: %v", err)
	}

	// Get connection string
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get Postgres DSN: %v", err)
	}

	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to Postgres: %v", err)
	}

	// Run migrations
	if err := runMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Load config
	cfgPath, err := findNearestEnvTest()
	if err != nil {
		t.Fatalf("Failed to find .env.test for tests: %v", err)
	}
	cfg, err := config.LoadAppConfig(slog.Default(), cfgPath)
	if err != nil {
		t.Fatalf("Failed to load app config for tests: %v", err)
	}

	// Override database URL with Testcontainers connection string
	cfg.DB.Url = dsn

	// Create test user with random credentials
	testUser, err := generateRandomTestUser()
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create services
	uow := infrarepo.NewUoW(db)
	logger := slog.Default()

	// Create auth service
	authStrategy := service.NewJWTAuthStrategy(uow, cfg.Jwt, logger)
	authService := service.NewAuthService(uow, authStrategy, logger)

	// Create currency converter
	currencyConverter := provider.NewStubCurrencyConverter()

	// Create services
	accountSvc := service.NewAccountService(uow, currencyConverter, logger)
	userSvc := service.NewUserService(uow, logger)

	// Initialize currency service
	currencyRegistry, err := currency.NewCurrencyRegistry(ctx)
	if err != nil {
		t.Fatalf("Failed to create currency registry for tests: %v", err)
	}

	// Load currency fixtures
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

	// Create app
	app := NewApp(accountSvc, userSvc, authService, currencySvc, cfg)
	log.SetOutput(io.Discard)

	return app, db, testUser, authService, cfg
}

// SetupSuite initializes the test suite with a real Postgres database
func (s *E2ETestSuiteWithDB) SetupSuite() {
	ctx := context.Background()

	// Start Postgres container
	pg, err := startPostgresContainer(ctx)
	s.Require().NoError(err)
	s.pgContainer = pg

	// Get connection string
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	s.Require().NoError(err)

	// Connect to database
	s.db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	s.Require().NoError(err)

	// Run migrations
	s.Require().NoError(runMigrations(s.db))

	// Load config
	cfgPath, err := findNearestEnvTest()
	s.Require().NoError(err)
	s.cfg, err = config.LoadAppConfig(slog.Default(), cfgPath)
	s.Require().NoError(err)

	// Override database URL with Testcontainers connection string
	s.cfg.DB.Url = dsn

	// Create test user with random credentials
	s.testUser, err = generateRandomTestUser()
	s.Require().NoError(err)

	// Create services
	uow := infrarepo.NewUoW(s.db)
	logger := slog.Default()

	// Create auth service
	authStrategy := service.NewJWTAuthStrategy(uow, s.cfg.Jwt, logger)
	s.authService = service.NewAuthService(uow, authStrategy, logger)

	// Create currency converter
	currencyConverter := provider.NewStubCurrencyConverter()

	// Create services
	accountSvc := service.NewAccountService(uow, currencyConverter, logger)
	userSvc := service.NewUserService(uow, logger)

	// Initialize currency service
	currencyRegistry, err := currency.NewCurrencyRegistry(ctx)
	s.Require().NoError(err)

	// Load currency fixtures
	_, filename, _, _ := runtime.Caller(0)
	fixturePath := filepath.Join(filepath.Dir(filename), "../internal/fixtures/currency/meta.csv")
	metas, err := fixturescurrency.LoadCurrencyMetaCSV(fixturePath)
	s.Require().NoError(err)
	for _, meta := range metas {
		s.Require().NoError(currencyRegistry.Register(meta))
	}
	currencySvc := service.NewCurrencyService(currencyRegistry, logger)

	// Create app
	s.app = NewApp(accountSvc, userSvc, s.authService, currencySvc, s.cfg)
	log.SetOutput(io.Discard)
}

// TearDownSuite cleans up the test suite resources
func (s *E2ETestSuiteWithDB) TearDownSuite() {
	ctx := context.Background()
	if s.pgContainer != nil {
		_ = s.pgContainer.Terminate(ctx)
	}
}

// generateTestToken makes an actual HTTP request to login and returns the JWT token
func (s *E2ETestSuiteWithDB) generateTestToken() string {
	// First, create the user in the database
	testUser := s.createTestUserInDB()

	// Make login request with the actual user credentials
	loginBody := fmt.Sprintf(`{"email":"%s","password":"password123"}`, testUser.Email)
	resp := s.makeRequest("POST", "/auth/login", loginBody, "")

	s.Require().Equal(200, resp.StatusCode)

	// Parse response to get token
	var loginResponse struct {
		Token string `json:"token"`
	}
	err := json.NewDecoder(resp.Body).Decode(&loginResponse)
	s.Require().NoError(err)
	s.Require().NotEmpty(loginResponse.Token)

	return loginResponse.Token
}

// makeRequest is a helper for making HTTP requests in tests
func (s *E2ETestSuiteWithDB) makeRequest(method, path, body, token string) *http.Response {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := s.app.Test(req)
	s.Require().NoError(err)
	return resp
}

// createTestUserInDB creates a unique test user in the database for each test
func (s *E2ETestSuiteWithDB) createTestUserInDB() *domain.User {
	// Create a unique test user for each test
	testUser, err := generateRandomTestUser()
	s.Require().NoError(err)

	uow := infrarepo.NewUoW(s.db)
	err = uow.Do(context.Background(), func(uow pkgrepo.UnitOfWork) error {
		userRepo, err := uow.GetRepository(reflect.TypeOf((*pkgrepo.UserRepository)(nil)).Elem())
		if err != nil {
			return err
		}
		return userRepo.(pkgrepo.UserRepository).Create(testUser)
	})
	s.Require().NoError(err)

	// Update the test user reference to the newly created one
	s.testUser = testUser
	return testUser
}
