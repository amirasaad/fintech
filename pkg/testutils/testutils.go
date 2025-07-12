package testutils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

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
	webapiaccount "github.com/amirasaad/fintech/webapi/account"
	webapiauth "github.com/amirasaad/fintech/webapi/auth"
	webapicurrency "github.com/amirasaad/fintech/webapi/currency"
	webapiuser "github.com/amirasaad/fintech/webapi/user"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// LoginUser makes an actual HTTP request to login and returns the JWT token
func LoginUser(app *fiber.App, testUser *domain.User) string {
	// Make login request with the actual user credentials
	loginBody := fmt.Sprintf(`{"identity":"%s","password":"password123"}`, testUser.Email)
	resp := MakeRequestWithApp(app, "POST", "/auth/login", loginBody, "")

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

// MakeRequest is a helper for making HTTP requests in tests
func MakeRequest(app *fiber.App, method, path, body, token string) *http.Response {
	return MakeRequestWithApp(app, method, path, body, token)
}

// MakeRequestWithApp is a helper for making HTTP requests with a standalone app (for non-suite tests)
func MakeRequestWithApp(app *fiber.App, method, path, body, token string) *http.Response {
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
	resp, err := app.Test(req, 1000000)
	if err != nil {
		panic(err) // For standalone tests, panic on error
	}
	return resp
}

// CreateTestUser creates a unique test user via the POST /user/ endpoint
func CreateTestUser(app *fiber.App) *domain.User {
	// Create a unique test user for each test
	testUser, err := generateRandomTestUser()
	if err != nil {
		panic(err) // For standalone tests, panic on error
	}

	// Create user via HTTP POST request
	createUserBody := fmt.Sprintf(`{"username":"%s","email":"%s","password":"password123"}`, testUser.Username, testUser.Email)
	resp := MakeRequestWithApp(app, "POST", "/user", createUserBody, "")

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

// AssertEqual is a helper for assertions in standalone tests
func AssertEqual(t *testing.T, expected, actual interface{}) {
	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

// SetupTestAppWithTestcontainers creates a test app using real Postgres via Testcontainers
func SetupTestAppWithTestcontainers(t *testing.T) (*fiber.App, *gorm.DB, *domain.User, *auth.AuthService, *config.AppConfig) {
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

	// Setup services
	authService, accountSvc, userSvc, currencySvc, err := setupServices(db, cfg)
	if err != nil {
		t.Fatalf("Failed to setup services: %v", err)
	}

	// Create app with higher rate limit for tests
	app := createTestApp(accountSvc, userSvc, authService, currencySvc, cfg)
	log.SetOutput(io.Discard)

	return app, db, testUser, authService, cfg
}

// createTestApp creates a test app with higher rate limits for testing
func createTestApp(
	accountSvc *account.AccountService,
	userSvc *userservice.UserService,
	authSvc *auth.AuthService,
	currencySvc *currencyservice.CurrencyService,
	cfg *config.AppConfig,
) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return apiutil.ProblemDetailsJSON(c, "Internal Server Error", err)
		},
	})

	// Add routes
	webapiaccount.AccountRoutes(app, accountSvc, authSvc, cfg)
	webapiuser.UserRoutes(app, userSvc, authSvc, cfg)
	webapiauth.AuthRoutes(app, authSvc)
	webapicurrency.CurrencyRoutes(app, currencySvc, authSvc, cfg)

	return app
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
func setupCurrencyRegistry(ctx context.Context) (*currency.CurrencyRegistry, error) {
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
func setupServices(db *gorm.DB, cfg *config.AppConfig) (*auth.AuthService, *account.AccountService, *userservice.UserService, *currencyservice.CurrencyService, error) {
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
	currencyRegistry, err := setupCurrencyRegistry(ctx)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	currencySvc := currencyservice.NewCurrencyService(currencyRegistry, logger)

	return authService, accountSvc, userSvc, currencySvc, nil
}
