package main_test

import (
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/amirasaad/fintech/webapi"
)

// TestMain runs before any tests and applies globally for all tests in the package.
func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	log.SetOutput(io.Discard)

	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestStartServer_RootRoute(t *testing.T) {
	app, _, _, _, _, _, _, _, _, _ := webapi.SetupTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Get / returns err, %s", err)
	}
	defer resp.Body.Close() // nolint: errcheck

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestProtectedRoute_Unauthorized(t *testing.T) {
	app, _, _, _, _, _, _, _, _, _ := webapi.SetupTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/account", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("GET /account returns err: %s", err)
	}
	defer resp.Body.Close() // nolint: errcheck
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected unauthorized or forbidden, got %d", resp.StatusCode)
	}
}

func TestNotFoundRoute(t *testing.T) {
	app, _, _, _, _, _, _, _, _, _ := webapi.SetupTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/doesnotexist", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("GET /doesnotexist returns err: %s", err)
	}
	defer resp.Body.Close() // nolint: errcheck
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestLoginRoute_BadRequest(t *testing.T) {
	app, _, _, _, _, _, _, _, _, _ := webapi.SetupTestApp(t)
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("POST /login returns err: %s", err)
	}
	defer resp.Body.Close() // nolint: errcheck
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
