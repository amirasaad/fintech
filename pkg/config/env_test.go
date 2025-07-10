package config

import (
	"os"
	"testing"

	"log/slog"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	// Test with environment variable set
	os.Setenv("TEST_VAR", "test_value") //nolint:errcheck
	defer os.Unsetenv("TEST_VAR")       //nolint:errcheck

	value := GetEnv("TEST_VAR", "default")
	assert.Equal(t, "test_value", value)

	// Test with environment variable not set
	value = GetEnv("NONEXISTENT_VAR", "default")
	assert.Equal(t, "default", value)
}

func TestIsEnvSet(t *testing.T) {
	// Test with environment variable set
	os.Setenv("TEST_VAR", "test_value") //nolint:errcheck
	defer os.Unsetenv("TEST_VAR")       //nolint:errcheck

	assert.True(t, IsEnvSet("TEST_VAR"))
	assert.False(t, IsEnvSet("NONEXISTENT_VAR"))
}

func TestGetEnvRequired(t *testing.T) {
	// Test with environment variable set
	os.Setenv("TEST_VAR", "test_value") //nolint:errcheck
	defer os.Unsetenv("TEST_VAR")       //nolint:errcheck

	value := GetEnvRequired("TEST_VAR")
	assert.Equal(t, "test_value", value)

	// Test with environment variable not set (should panic)
	assert.Panics(t, func() {
		GetEnvRequired("NONEXISTENT_VAR") //nolint:errcheck
	})
}

func TestLoadEnv(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Test loading environment variables (should not panic)
	assert.NotPanics(t, func() {
		LoadEnv(logger)
	})
}
