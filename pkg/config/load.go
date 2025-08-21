package config

import (
	"log/slog"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

func Load(envFilePath ...string) (*App, error) {
	logger := slog.Default()
	logger.Info("Loading environment variables")

	// If no specific paths provided, try default .env
	if len(envFilePath) == 0 {
		logger.Debug("No environment file specified, trying default .env")
		if err := godotenv.Load(); err != nil {
			logger.Warn("No .env file found in current directory")
		}
		return loadFromEnv()
	}

	// Try each provided path until we find a valid one
	for _, path := range envFilePath {
		logger.Debug("Looking for environment file", "path", path)
		foundPath, err := FindEnvTest(path)
		if err != nil {
			logger.Debug("Environment file not found", "path", path, "error", err)
			continue
		}

		logger.Info("Loading environment from file", "path", foundPath)
		if err := godotenv.Load(foundPath); err != nil {
			logger.Error("Failed to load environment file", "path", foundPath, "error", err)
			continue
		}

		// Successfully loaded a file, proceed with config loading
		return loadFromEnv()
	}

	// No valid environment files found, try default .env as fallback
	logger.Info("No valid environment files found, using default .env")
	if err := godotenv.Load(); err != nil {
		logger.Warn("No .env file found in current directory")
	}
	return loadFromEnv()
}

func loadFromEnv() (*App, error) {
	var cfg App
	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, err
	}

	// Set default values if not set
	if cfg.Env == "" {
		cfg.Env = "development"
	}

	logger := slog.Default()
	logger.Info("Environment variables loaded from .env file")
	logger.Info("App config loaded",
		"env", cfg.Env,
		"rate_limit_max_requests", cfg.RateLimit.MaxRequests,
		"rate_limit_window", cfg.RateLimit.Window,
		"db", maskValue(cfg.DB.Url),
		"auth_strategy", cfg.Auth.Strategy,
		"auth_jwt_expiry", cfg.Auth.Jwt.Expiry,
		"exchange_cache_ttl", cfg.ExchangeRateCache.TTL,
		"exchange_api_url", cfg.ExchangeRateAPIProviders.ExchangeRateApi.ApiUrl,
		"exchange_api_key", maskValue(cfg.ExchangeRateAPIProviders.ExchangeRateApi.ApiKey),
	)
	return &cfg, nil
}

func maskValue(key string) string {
	if len(key) <= 6 {
		return "****"
	}
	return key[:2] + "****" + key[len(key)-4:]
}
