package handler

import (
	"log"
	"log/slog"
	"net/http"

	"github.com/amirasaad/fintech/infra/initializer"
	"github.com/amirasaad/fintech/pkg/app"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/webapi"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
)

// Handler_api_index_go is the main entry point of the application.
// Think of it like the main() method
func Handler_api_index_go(w http.ResponseWriter, r *http.Request) {
	// This is needed to set the proper request path in `*fiber.Ctx`
	r.RequestURI = r.URL.String()

	handler().ServeHTTP(w, r)
}

// building the fiber application
func handler() http.HandlerFunc {
	logger := slog.Default()
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load application configuration", "error", err)
		log.Fatal(err)
	}

	// Initialize all dependencies
	deps, err := initializer.InitializeDependencies(cfg)
	if err != nil {
		logger.Error("Failed to initialize application dependencies", "error", err)
		log.Fatal(err)
	}

	// Initialize the application
	a := app.New(deps, cfg)

	// Setup Fiber app with the initialized application
	fiberApp := webapi.SetupApp(a)

	// Return the Fiber app as an http.Handler
	return adaptor.FiberApp(fiberApp)
}
