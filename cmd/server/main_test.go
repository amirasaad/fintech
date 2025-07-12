package main_test

import (
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/webapi"
	"github.com/amirasaad/fintech/webapi/testutils"
	"github.com/stretchr/testify/suite"
)

// TestMain runs before any tests and applies globally for all tests in the package.
func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	log.SetOutput(io.Discard)

	exitVal := m.Run()
	os.Exit(exitVal)
}

type MainTestSuite struct {
	testutils.E2ETestSuite
}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}

func (s *MainTestSuite) TestStartServer_RootRoute() {
	// Create a minimal app for testing
	app := webapi.NewApp(nil, nil, nil, nil, &config.AppConfig{})

	// Use httptest to make the request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		s.T().Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.T().Fatalf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func (s *MainTestSuite) TestProtectedRoute_Unauthorized() {
	resp := s.MakeRequest(http.MethodGet, "/account", "", "")
	defer resp.Body.Close() //nolint: errcheck
	if resp.StatusCode != http.StatusUnauthorized {
		s.T().Fatalf("expected unauthorized or forbidden, got %d", resp.StatusCode)
	}
}

func (s *MainTestSuite) TestNotFoundRoute() {
	resp := s.MakeRequest(http.MethodGet, "/doesnotexist", "", "")
	defer resp.Body.Close() //nolint: errcheck
	if resp.StatusCode != http.StatusNotFound {
		s.T().Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func (s *MainTestSuite) TestLoginRoute_BadRequest() {
	resp := s.MakeRequest(http.MethodPost, "/auth/login", "", "")
	defer resp.Body.Close() //nolint: errcheck
	if resp.StatusCode != http.StatusBadRequest {
		s.T().Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
