package config

import (
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// LoadEnv loads environment variables from .env file
// If .env file is not found, it logs a warning and continues
func LoadEnv(logger *slog.Logger) {
	if err := godotenv.Load(); err != nil {
		logger.Warn("No .env file found, using system environment variables")
	} else {
		logger.Info("Environment variables loaded from .env file")
	}
}

// GetEnv retrieves an environment variable with a default value
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvRequired retrieves a required environment variable
// Panics if the environment variable is not set
func GetEnvRequired(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic("Required environment variable " + key + " is not set")
	}
	return value
}

// IsEnvSet checks if an environment variable is set
func IsEnvSet(key string) bool {
	return os.Getenv(key) != ""
}

// GetEnvAsInt retrieves an environment variable as int with a default value
func GetEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetEnvAsBool retrieves an environment variable as bool with a default value
func GetEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// GetEnvAsDuration retrieves an environment variable as time.Duration with a default value
func GetEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
