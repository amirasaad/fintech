package testutils

import (
	"context"
	"io"
	"log"
	"log/slog"
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
	"github.com/amirasaad/fintech/pkg/apiutil"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/service/account"
	"github.com/amirasaad/fintech/pkg/service/auth"
	currencyservice "github.com/amirasaad/fintech/pkg/service/currency"
	userservice "github.com/amirasaad/fintech/pkg/service/user"
	"github.com/amirasaad/fintech/webapi"

	"github.com/gofiber/fiber/v2"
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

// E2ETestSuite provides a test suite with a real Postgres database using Testcontainers
type E2ETestSuite struct {
	suite.Suite
	pgContainer *tcpostgres.PostgresContainer
	db          *gorm.DB
	app         *fiber.App
	cfg         *config.AppConfig
}

func (s *E2ETestSuite) BeforeTest() {
	s.T().Parallel()
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
func (s *E2ETestSuite) startPostgresContainer(ctx context.Context) (*tcpostgres.PostgresContainer, error) {
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
func (s *E2ETestSuite) runMigrations(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	driver, err := migratepostgres.WithInstance(sqlDB, &migratepostgres.Config{})
	if err != nil {
		return err
	}

	_, filename, _, _ := runtime.Caller(0)
	migrationsPath := filepath.Join(filepath.Dir(filename), "../../internal/migrations")

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

// setupCurrencyRegistry initializes the currency registry with test fixtures
func (s *E2ETestSuite) setupCurrencyRegistry(ctx context.Context) (*currency.CurrencyRegistry, error) {
	currencyRegistry, err := currency.NewCurrencyRegistry(ctx)
	if err != nil {
		return nil, err
	}

	// Load currency fixtures
	_, filename, _, _ := runtime.Caller(0)
	fixturePath := filepath.Join(filepath.Dir(filename), "../../internal/fixtures/currency/meta.csv")
	metas, err := fixturescurrency.LoadCurrencyMetaCSV(fixturePath)
	if err != nil {
		return nil, err
	}

	for _, meta := range metas {
		if err := currencyRegistry.Register(meta); err != nil {
			return nil, err
		}
	}

	return currencyRegistry, nil
}

// setupServices creates all required services for testing
func (s *E2ETestSuite) setupServices(db *gorm.DB, cfg *config.AppConfig) (*auth.AuthService, *account.AccountService, *userservice.UserService, *currencyservice.CurrencyService, error) {
	uow := infrarepo.NewUoW(db)
	logger := slog.Default()

	// Create auth service
	authStrategy := auth.NewJWTAuthStrategy(uow, cfg.Jwt, logger)
	authService := auth.NewAuthService(uow, authStrategy, logger)

	// Create currency converter
	currencyConverter := provider.NewStubCurrencyConverter()

	// Create services
	accountSvc := account.NewAccountService(uow, currencyConverter, logger)
	userSvc := userservice.NewUserService(uow, logger)

	// Initialize currency service
	ctx := context.Background()
	currencyRegistry, err := s.setupCurrencyRegistry(ctx)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	currencySvc := currencyservice.NewCurrencyService(currencyRegistry, logger)

	return authService, accountSvc, userSvc, currencySvc, nil
}

// createTestApp creates a minimal Fiber app for testing without importing webapi packages
func (s *E2ETestSuite) createTestApp(
	accountSvc *account.AccountService,
	userSvc *userservice.UserService,
	authSvc *auth.AuthService,
	currencySvc *currencyservice.CurrencyService,
	cfg *config.AppConfig,
) *fiber.App {
	app := webapi.NewApp(
		accountSvc, userSvc, authSvc, currencySvc, cfg,
	)

	return app
}

// SetupSuite initializes the test suite with a real Postgres database
func (s *E2ETestSuite) SetupSuite() {
	ctx := context.Background()

	// Start Postgres container
	pg, err := s.startPostgresContainer(ctx)
	s.Require().NoError(err)
	s.pgContainer = pg

	// Get connection string
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	s.Require().NoError(err)

	// Connect to database
	s.db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	s.Require().NoError(err)

	// Run migrations
	s.Require().NoError(s.runMigrations(s.db))

	// Load config
	cfgPath, err := findNearestEnvTest()
	s.Require().NoError(err)
	s.cfg, err = config.LoadAppConfig(slog.Default(), cfgPath)
	s.Require().NoError(err)

	// Override database URL with Testcontainers connection string
	s.cfg.DB.Url = dsn

	// Setup services
	authService, accountSvc, userSvc, currencySvc, err := s.setupServices(s.db, s.cfg)
	s.Require().NoError(err)

	// Create test app
	s.app = s.createTestApp(accountSvc, userSvc, authService, currencySvc, s.cfg)
	log.SetOutput(io.Discard)
}

// TearDownSuite cleans up the test suite resources
func (s *E2ETestSuite) TearDownSuite() {
	ctx := context.Background()
	if s.pgContainer != nil {
		_ = s.pgContainer.Terminate(ctx)
	}
}

// MakeRequest is a helper for making HTTP requests in tests
func (s *E2ETestSuite) MakeRequest(method, path, body, token string) *http.Response {
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
	resp, err := s.app.Test(req, 1000000)
	if err != nil {
		panic(err) // For standalone tests, panic on error
	}
	return resp
}

// LoginUser makes an actual HTTP request to login and returns the JWT token
func (s *E2ETestSuite) LoginUser(testUser *domain.User) string {
	// Make login request with the actual user credentials
	loginBody := fmt.Sprintf(`{"identity":"%s","password":"password123"}`, testUser.Email)
	resp := s.MakeRequest("POST", "/auth/login", loginBody, "")

	// Parse response to get token and log response
	var response apiutil.Response
	err := json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		panic(err) // For standalone tests, panic on error
	}

	// Handle the data field which can be map[string]interface{} or map[string]string
	var token string
	if dataMap, ok := response.Data.(map[string]any); ok {
		if tokenInterface, exists := dataMap["token"]; exists {
			token = tokenInterface.(string)
		}
	} else if dataMap, ok := response.Data.(map[string]string); ok {
		token = dataMap["token"]
	}

	if token == "" {
		panic("No token found in response")
	}
	return token
}

// CreateTestUser creates a unique test user via the POST /user/ endpoint
func (s *E2ETestSuite) CreateTestUser() *domain.User {
	// Create a unique test user for each test
	testUser, err := generateRandomTestUser()
	if err != nil {
		panic(err) // For standalone tests, panic on error
	}

	// Create user via HTTP POST request
	createUserBody := fmt.Sprintf(`{"username":"%s","email":"%s","password":"password123"}`, testUser.Username, testUser.Email)
	resp := s.MakeRequest("POST", "/user", createUserBody, "")

	if resp.StatusCode != 201 {
		panic(fmt.Sprintf("Expected 201 Created for user creation, got %d", resp.StatusCode))
	}

	// Parse response to get the created user
	var response apiutil.Response
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		panic(err)
	}

	// Extract user data from response
	var createdUser *domain.User
	if userData, ok := response.Data.(map[string]any); ok {
		// Convert the response data back to a domain.User
		// This assumes the response contains the user data
		userIDStr, ok := userData["id"].(string)
		if !ok {
			panic("User ID should be present in response")
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			panic(err)
		}

		createdUser = &domain.User{
			ID:       userID,
			Username: testUser.Username,
			Email:    testUser.Email,
			Password: testUser.Password, // Note: this might not be returned in the response
		}
	} else {
		// Fallback: use the original test user with the generated data
		createdUser = testUser
	}

	return createdUser
}
