package main

import (
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/infra/initializer"
	"github.com/amirasaad/fintech/pkg/app"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/webapi"
	log "github.com/charmbracelet/log"
)

// @title Fintech API
// @version 1.0.0
// @description Fintech API documentation
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email fiber@swagger.io
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/MIT
// @host localhost:3000
// @BasePath /
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description "Enter your Bearer token in the format: `Bearer {token}`"
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Load configuration
	logger := slog.Default()
	cfg, err := config.Load(".env")

	if err != nil {
		return fmt.Errorf("failed to load application configuration: %w", err)
	}

	// Initialize all dependencies
	deps, err := initializer.InitializeDependencies(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize dependencies: %w", err)
	}

	logger.Info(
		"starting server",
		"env", cfg.Env,
		"scheme", cfg.Server.Scheme,
		"host", cfg.Server.Host,
		"port", cfg.Server.Port,
	)

	// Create and start the application
	app := app.New(deps, cfg)

	// Setup Fiber app with all routes and middleware
	fiberApp := webapi.SetupApp(app)

	// Start the server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logger.Info("Starting server",
		"env", cfg.Env,
		"address", addr,
		"scheme", cfg.Server.Scheme,
	)

	return fiberApp.Listen(addr)
}
