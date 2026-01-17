package handler

import (
	"encoding/base64"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

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
	if err := ensureKafkaCAFile(logger); err != nil {
		logger.Error("Failed to prepare Kafka CA file", "error", err)
		log.Fatal(err)
	}
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

const (
	kafkaTLSCAEnv    = "EVENT_BUS_KAFKA_TLS_CA_PEM"
	kafkaTLSCAEnvB64 = "EVENT_BUS_KAFKA_TLS_CA_PEM_B64"
	kafkaTLSCAFile   = "EVENT_BUS_KAFKA_TLS_CA_FILE"
)

func ensureKafkaCAFile(logger *slog.Logger) error {
	pem := strings.TrimSpace(os.Getenv(kafkaTLSCAEnv))
	pemB64 := strings.TrimSpace(os.Getenv(kafkaTLSCAEnvB64))
	if pem == "" && pemB64 == "" {
		return nil
	}

	if pem == "" {
		decoded, err := base64.StdEncoding.DecodeString(pemB64)
		if err != nil {
			return fmt.Errorf("decode kafka ca pem: %w", err)
		}
		pem = string(decoded)
	}

	pem = strings.ReplaceAll(pem, "\\n", "\n")
	if strings.TrimSpace(pem) == "" {
		return fmt.Errorf("kafka ca pem is empty")
	}

	path := strings.TrimSpace(os.Getenv(kafkaTLSCAFile))
	if path == "" {
		tmpfile, err := os.CreateTemp("", "fintech-kafka-ca-*.pem")
		if err != nil {
			return fmt.Errorf("create temp kafka ca file: %w", err)
		}
		if err := tmpfile.Close(); err != nil {
			return fmt.Errorf("close temp kafka ca file: %w", err)
		}
		path = tmpfile.Name()
		if err := os.Setenv(kafkaTLSCAFile, path); err != nil {
			return fmt.Errorf("set kafka ca file env: %w", err)
		}
	}

	if err := os.WriteFile(path, []byte(pem), 0600); err != nil {
		return fmt.Errorf("write kafka ca file: %w", err)
	}
	if logger != nil {
		logger.Info("Kafka CA file written", "path", path)
	}

	return nil
}
