package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/amirasaad/fintech/infra/initializer"
	"github.com/amirasaad/fintech/pkg/app"
	"github.com/amirasaad/fintech/webapi"
	"github.com/charmbracelet/lipgloss"
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
	logger := setupLogger()

	// Initialize all dependencies
	deps, cfg, err := initializer.InitializeDependencies(logger)
	if err != nil {
		return fmt.Errorf("failed to initialize dependencies: %w", err)
	}

	logger.Info(
		"starting server",
		"env", cfg.Env,
		"port", cfg.Port,
	)

	// Create and start the application
	appInstance := app.New(deps, cfg)

	// Setup Fiber app with all routes and middleware
	fiberApp := webapi.SetupApp(appInstance)

	// Start the server
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	logger.Info("Starting server",
		"env", cfg.Env,
		"address", addr,
		"scheme", cfg.Scheme,
	)

	return fiberApp.Listen(addr)
}

func setupLogger() *slog.Logger {
	// Create a new logger with a custom style
	// Define color styles for different log levels
	styles := log.DefaultStyles()
	infoTxtColor := lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"}
	warnTxtColor := lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"}
	errorTxtColor := lipgloss.AdaptiveColor{Light: "#FF6B6B", Dark: "#FF6B6B"}
	debugTxtColor := lipgloss.AdaptiveColor{Light: "#7E57C2", Dark: "#7E57C2"}

	// Customize the style for each log level
	// Error level styling
	styles.Levels[log.ErrorLevel] = lipgloss.NewStyle().
		SetString("❌ ERROR").
		Bold(true).
		Padding(0, 1).
		Foreground(errorTxtColor)

	// Info level styling
	styles.Levels[log.InfoLevel] = lipgloss.NewStyle().
		SetString("ℹ️  INFO").
		Bold(true).
		Padding(0, 1).
		Foreground(infoTxtColor)

	// Warn level styling
	styles.Levels[log.WarnLevel] = lipgloss.NewStyle().
		SetString("⚠️  WARN").
		Bold(true).
		Padding(0, 1).
		Foreground(warnTxtColor)

	// Debug level styling
	styles.Levels[log.DebugLevel] = lipgloss.NewStyle().
		SetString("🐛 DEBUG").
		Bold(true).
		Padding(0, 1).
		Foreground(debugTxtColor)

	// Create a new logger with the custom styles
	logger := log.NewWithOptions(os.Stdout, log.Options{
		ReportCaller:    false,
		ReportTimestamp: true,
		TimeFormat:      time.Kitchen,
		Level:           log.DebugLevel,
		Prefix:          "[fintech]",
	})

	logger.SetStyles(styles) // Convert to slog.Logger
	slogger := slog.New(logger)
	slog.SetDefault(slogger)

	return slogger
}
