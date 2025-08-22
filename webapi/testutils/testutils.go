package testutils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"time"

	"github.com/amirasaad/fintech/pkg/app"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/registry"

	"github.com/amirasaad/fintech/infra/eventbus"
	mockexchangerate "github.com/amirasaad/fintech/infra/provider/mockexchangerate"
	"github.com/amirasaad/fintech/infra/provider/mockpayment"
	infrarepo "github.com/amirasaad/fintech/infra/repository"
	fixturescurrency "github.com/amirasaad/fintech/internal/fixtures/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/webapi"
	"github.com/amirasaad/fintech/webapi/common"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // required for file-based migrations
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
	cfg         *config.App
}

// BeforeEachTest runs before each test in the E2ETestSuite. It enables parallel test execution.
func (s *E2ETestSuite) BeforeEachTest() {
	s.T().Parallel()
}

// SetupSuite initializes the test suite with a real Postgres database
func (s *E2ETestSuite) SetupSuite() {
	ctx := context.Background()

	// Start Postgres container
	pg, err := tcpostgres.Run(
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
	s.Require().NoError(err)
	s.pgContainer = pg

	// Get connection string and connect to database
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	s.Require().NoError(err)

	s.db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	s.Require().NoError(err)

	// Run migrations
	sqlDB, err := s.db.DB()
	s.Require().NoError(err)

	driver, err := migratepostgres.WithInstance(sqlDB, &migratepostgres.Config{})
	s.Require().NoError(err)

	_, filename, _, _ := runtime.Caller(0)
	migrationsPath := filepath.Join(filepath.Dir(filename), "../../internal/migrations")

	m, err := migrate.NewWithDatabaseInstance("file://"+migrationsPath, "postgres", driver)
	s.Require().NoError(err)

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		s.Require().NoError(err)
	}

	// Load config
	envTest, err := config.FindEnvTest(".env.test")
	s.Require().NoError(err)
	s.cfg, err = config.Load(envTest)
	s.Require().NoError(err)
	s.cfg.DB.Url = dsn

	// Setup services and app
	s.setupApp()
	// log.SetOutput(io.Discard)
}

// TearDownSuite cleans up the test suite resources
func (s *E2ETestSuite) TearDownSuite() {
	ctx := context.Background()
	if s.pgContainer != nil {
		_ = s.pgContainer.Terminate(ctx)
	}
}

// setupApp creates all services and the test app,
// using Redis as the event bus via testcontainers-go.
func (s *E2ETestSuite) setupApp() {
	s.T().Helper()
	// Create deps with debug logging
	uow := infrarepo.NewUoW(s.db)
	// Enable debug logging
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Setup currency service
	ctx := context.Background()
	currencyRegistry, err := registry.NewBuilder().
		WithName("test_currency").
		WithRedis("").
		WithCache(100, time.Minute).
		BuildRegistry()
	if err != nil {
		panic(fmt.Errorf("failed to create test currency registry provider: %w", err))
	}

	// Load currency fixtures
	_, filename, _, _ := runtime.Caller(0)
	fixturePath := filepath.Join(
		filepath.Dir(filename),
		"../../internal/fixtures/currency/meta.csv",
	)
	metas, err := fixturescurrency.LoadCurrencyMetaCSV(fixturePath)
	s.Require().NoError(err)

	for _, meta := range metas {
		s.Require().NoError(currencyRegistry.Register(ctx, meta))
	}

	// Start Redis container
	redisContainer, err := testcontainers.GenericContainer(
		ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        "redis:7-alpine",
				ExposedPorts: []string{"6379/tcp"},
				WaitingFor: wait.ForListeningPort(
					"6379/tcp",
				).WithStartupTimeout(10 * time.Second),
			},
			Started: true,
		},
	)
	s.Require().NoError(err)

	endpoint, err := redisContainer.Endpoint(ctx, "")
	s.Require().NoError(err)

	// Setup Redis EventBus
	eventBus, err := eventbus.NewWithRedis("redis://"+endpoint, logger)
	s.Require().NoError(err)

	// Store Redis container for cleanup at the end of this test
	s.T().Cleanup(func() {
		_ = redisContainer.Terminate(ctx)
	})

	// Create registry providers for each service with in-memory storage
	mainReg, err := registry.NewBuilder().
		WithName("test").
		WithRedis(""). // Empty URL for in-memory
		WithCache(100, time.Minute).
		BuildRegistry()
	if err != nil {
		panic(fmt.Errorf("failed to create test main registry provider: %w", err))
	}
	mainRegistry, ok := mainReg.(*registry.Enhanced)
	if !ok {
		panic("main registry is not of type *registry.Enhanced")
	}

	// Create currency registry
	currencyReg, err := registry.NewBuilder().
		WithName("test_currency").
		WithRedis("").
		WithCache(100, time.Minute).
		BuildRegistry()
	if err != nil {
		panic(fmt.Errorf("failed to create test currency registry provider: %w", err))
	}
	currencyRegistry, ok = currencyReg.(*registry.Enhanced)
	if !ok {
		panic("currency registry is not of type *registry.Enhanced")
	}

	// Create checkout registry
	checkoutReg, err := registry.NewBuilder().
		WithName("test_checkout").
		WithRedis("").
		WithCache(100, time.Minute).
		BuildRegistry()
	if err != nil {
		panic(fmt.Errorf("failed to create test checkout registry provider: %w", err))
	}
	checkoutRegistry, ok := checkoutReg.(*registry.Enhanced)
	if !ok {
		panic("checkout registry is not of type *registry.Enhanced")
	}

	// Create exchange rate registry
	exchangeRateReg, err := registry.NewBuilder().
		WithName("test_exchange_rate").
		WithRedis("").
		WithCache(100, time.Minute).
		BuildRegistry()
	if err != nil {
		panic(fmt.Errorf("failed to create test exchange rate registry provider: %w", err))
	}
	exchangeRateRegistry, ok := exchangeRateReg.(*registry.Enhanced)
	if !ok {
		panic("exchange rate registry is not of type *registry.Enhanced")
	}
	exchangeRateProvider := mockexchangerate.NewMockExchangeRate()
	mockPaymentProvider := mockpayment.NewMockPaymentProvider()

	deps := &app.Deps{
		RegistryProvider:     mainRegistry,
		CurrencyRegistry:     currencyRegistry,
		CheckoutRegistry:     checkoutRegistry,
		ExchangeRateRegistry: exchangeRateRegistry,
		ExchangeRateProvider: exchangeRateProvider,
		PaymentProvider:      mockPaymentProvider,
		Uow:                  uow,
		EventBus:             eventBus,
		Logger:               logger,
	}

	// Create test app
	s.app = webapi.SetupApp(app.New(
		deps,
		s.cfg,
	))
}

// MakeRequest is a helper for making HTTP requests in tests
func (s *E2ETestSuite) MakeRequest(
	method, path, body, token string,
) *http.Response {
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
		s.T().Fatal(err)
	}
	return resp
}

// CreateTestUser creates a unique test user via the POST /user/ endpoint
func (s *E2ETestSuite) CreateTestUser() *domain.User {
	randomID := uuid.New().String()[:8]
	username := fmt.Sprintf("testuser_%s", randomID)
	email := fmt.Sprintf("test_%s@example.com", randomID)

	// Create user via HTTP POST request
	createUserBody := fmt.Sprintf(
		`{"username":"%s","email":"%s","password":"password123"}`,
		username,
		email,
	)
	resp := s.MakeRequest("POST", "/user", createUserBody, "")

	if resp.StatusCode != 201 {
		// Read the response body for more details
		body, _ := io.ReadAll(resp.Body)
		s.T().Logf("User creation failed with status %d.", resp.StatusCode)
		s.T().Logf("Response body: %s", string(body))
		s.T().Fatalf("Expected 201 Created for user creation, got %d", resp.StatusCode)
	}

	// Parse response to get the created user
	var response common.Response
	err := json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		s.T().Fatal(err)
	}

	// Extract user data from response
	if userData, ok := response.Data.(map[string]any); ok {
		userIDStr, ok := userData["id"].(string)
		if !ok {
			s.T().Fatalf("User ID should be present in response")
		}

		userID, parseErr := uuid.Parse(userIDStr)
		if parseErr != nil {
			s.T().Fatalf("User ID should be a valid UUID")
		}

		return &domain.User{
			ID:       userID,
			Username: username,
			Email:    email,
			Password: "password123",
		}
	}

	// Fallback: create user directly
	testUser, err := user.New(username, email, "password123")
	if err != nil {
		s.T().Fatalf("Failed to create user: %v", err)
	}
	return testUser
}

// LoginUser makes an actual HTTP request to login and returns the JWT token
func (s *E2ETestSuite) LoginUser(testUser *domain.User) string {
	loginBody := fmt.Sprintf(`{"identity":"%s","password":"%s"}`, testUser.Email, testUser.Password)
	resp := s.MakeRequest("POST", "/auth/login", loginBody, "")

	var response common.Response
	err := json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		s.T().Fatal(err)
	}

	// Extract token from response
	var token string
	if dataMap, ok := response.Data.(map[string]any); ok {
		if tokenInterface, exists := dataMap["token"]; exists {
			token = tokenInterface.(string)
		}
	} else if dataMap, ok := response.Data.(map[string]string); ok {
		token = dataMap["token"]
	} else if tokenString, ok := response.Data.(string); ok {
		token = tokenString
	}

	s.T().Logf("Extracted token: %s", token)
	if token == "" {
		s.T().Fatalf("No token found in response")
	}
	return token
}
