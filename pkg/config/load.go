package config

import (
	"log/slog"
	"regexp"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

func Load(envFilePath ...string) (*App, error) {
	var err error
	logger := slog.Default()
	if len(envFilePath) > 0 && envFilePath[0] != "" {
		err = godotenv.Load(envFilePath[0])
	} else {
		err = godotenv.Load()
	}

	if err != nil {
		logger.Warn("No .env file found or specified, using system environment variables")
	} else {
		logger.Info("Environment variables loaded from .env file")
	}
	var cfg App
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	logger.Info("App config loaded",
		"db", cfg.DB.Url,
		"auth_strategy", cfg.Auth.Strategy,
		"auth_jwt_expiry", cfg.Auth.Jwt.Expiry,
		"exchange_cache_ttl", cfg.Exchange.CacheTTL,
		"exchange_api_url", maskApiKeyInUrl(cfg.Exchange.ApiUrl),
		"exchange_api_key", maskValue(cfg.Exchange.ApiKey),
		"rate_limit_max_requests", cfg.RateLimit.MaxRequests,
		"rate_limit_window", cfg.RateLimit.Window,
	)
	return &cfg, nil
}

func maskValue(key string) string {
	if len(key) <= 6 {
		return "****"
	}
	return key[:2] + "****" + key[len(key)-4:]
}

func maskApiKeyInUrl(url string) string {
	// Mask /v6/<key> in path
	re := regexp.MustCompile(`(v6/)[^/]+`)
	masked := re.ReplaceAllString(url, `${1}[MASKED]`)
	// Mask api_key in query string
	qre := regexp.MustCompile(`([?&]api_key=)[^&]+`)
	masked = qre.ReplaceAllString(masked, `${1}[MASKED]`)
	return masked
}
