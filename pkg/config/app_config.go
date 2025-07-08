package config

import (
	"log/slog"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type DBConfig struct {
	Url string `envconfig:"URL" default:"postgres://postgres:password@localhost:5432/fintech?sslmode=disable"`
}

type AuthConfig struct {
	Strategy string `envconfig:"STRATEGY" default:"jwt"`
}
type JwtConfig struct {
	Secret string        `envconfig:"SECRET" required:"true"`
	Expiry time.Duration `envconfig:"EXPIRY" default:"24h"`
}

type ExchangeRateConfig struct {
	ApiKey            string        `envconfig:"API_KEY"`
	ApiUrl            string        `envconfig:"API_URL" default:"https://api.exchangerate-api.com/v4/latest"`
	CacheTTL          time.Duration `envconfig:"CACHE_TTL" default:"15m"`
	HTTPTimeout       time.Duration `envconfig:"HTTP_TIMEOUT" default:"10s"`
	MaxRetries        int           `envconfig:"MAX_RETRIES" default:"3"`
	RequestsPerMinute int           `envconfig:"REQUESTS_PER_MINUTE" default:"60"`
	BurstSize         int           `envconfig:"BURST_SIZE" default:"10"`
	EnableFallback    bool          `envconfig:"ENABLE_FALLBACK" default:"true"`
	FallbackTTL       time.Duration `envconfig:"FALLBACK_TTL" default:"1h"`
}

type AppConfig struct {
	DB       DBConfig           `envconfig:"DATABASE"`
	Auth     AuthConfig         `envconfig:"AUTH"`
	Jwt      JwtConfig          `envconfig:"JWT"`
	Exchange ExchangeRateConfig `envconfig:"EXCHANGE_RATE"`
}

func LoadAppConfig(logger *slog.Logger) (*AppConfig, error) {
	if err := godotenv.Load(); err != nil {
		logger.Warn("No .env file found, using system environment variables")
	} else {
		logger.Info("Environment variables loaded from .env file")
	}
	var cfg AppConfig
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	logger.Info("App config loaded", "db", cfg.DB.Url, "jwt_expiry", cfg, "exchange_cache_ttl", cfg.Exchange.CacheTTL)
	return &cfg, nil
}
