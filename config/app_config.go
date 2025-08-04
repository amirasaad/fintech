package config

import (
	"log/slog"
	"regexp"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type DBConfig struct {
	Url string `envconfig:"URL"`
}

type AuthConfig struct {
	Strategy string `envconfig:"STRATEGY" default:"jwt"`
}
type JwtConfig struct {
	Secret string        `envconfig:"SECRET_KEY" required:"true"`
	Expiry time.Duration `envconfig:"EXPIRY" default:"24h"`
}

type RedisConfig struct {
	URL string `envconfig:"REDIS_URL" default:"redis://localhost:6379/0"`
}

type RateLimitConfig struct {
	MaxRequests int           `envconfig:"MAX_REQUESTS" default:"100"`
	Window      time.Duration `envconfig:"WINDOW" default:"1m"`
}

type Stripe struct {
	ApiKey        string `envconfig:"API_KEY"`
	SigningSecret string `envconfig:"SIGNING_SECRET"`
}

type PaymentProviders struct {
	Stripe Stripe `envconfig:"STRIPE"`
}

type ExchangeRateConfig struct {
	ApiKey            string        `envconfig:"API_KEY"`
	ApiUrl            string        `envconfig:"API_URL" default:""`
	CacheTTL          time.Duration `envconfig:"CACHE_TTL" default:"15m"`
	HTTPTimeout       time.Duration `envconfig:"HTTP_TIMEOUT" default:"10s"`
	MaxRetries        int           `envconfig:"MAX_RETRIES" default:"3"`
	RequestsPerMinute int           `envconfig:"REQUESTS_PER_MINUTE" default:"60"`
	BurstSize         int           `envconfig:"BURST_SIZE" default:"10"`
	EnableFallback    bool          `envconfig:"ENABLE_FALLBACK" default:"true"`
	FallbackTTL       time.Duration `envconfig:"FALLBACK_TTL" default:"1h"`
	CachePrefix       string        `envconfig:"CACHE_PREFIX" default:"exr:rate:"`
	CacheUrl          string        `envconfig:"CACHE_URL"`
}

type AppConfig struct {
	Env              string             `envconfig:"APP_ENV" default:"development"`
	Scheme           string             `envconfig:"APP_SCHEME" default:"https"`
	Host             string             `envconfig:"APP_HOST" default:"localhost"`
	Port             int                `envconfig:"APP_PORT" default:"3000"`
	DB               DBConfig           `envconfig:"DATABASE"`
	Auth             AuthConfig         `envconfig:"AUTH"`
	Jwt              JwtConfig          `envconfig:"JWT"`
	Exchange         ExchangeRateConfig `envconfig:"EXCHANGE_RATE"`
	Redis            RedisConfig        `envconfig:"REDIS"`
	RateLimit        RateLimitConfig    `envconfig:"RATE_LIMIT"`
	PaymentProviders PaymentProviders   `envconfig:"PAYMENT_PROVIDER"`
}

func maskApiKey(key string) string {
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

func LoadAppConfig(logger *slog.Logger, envFilePath ...string) (*AppConfig, error) {
	var err error
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
	var cfg AppConfig
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	logger.Info("App config loaded",
		"db", cfg.DB.Url,
		"jwt_expiry", cfg.Jwt.Expiry,
		"exchange_cache_ttl", cfg.Exchange.CacheTTL,
		"exchange_api_url", maskApiKeyInUrl(cfg.Exchange.ApiUrl),
		"exchange_api_key", maskApiKey(cfg.Exchange.ApiKey),
		"rate_limit_max_requests", cfg.RateLimit.MaxRequests,
		"rate_limit_window", cfg.RateLimit.Window,
	)
	return &cfg, nil
}
