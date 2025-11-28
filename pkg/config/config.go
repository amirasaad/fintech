package config

import (
	"time"
)

type DB struct {
	Url string `envconfig:"URL"`
}

type Jwt struct {
	Secret string        `envconfig:"SECRET" required:"true"`
	Expiry time.Duration `envconfig:"EXPIRY" default:"24h"`
}
type Auth struct {
	Strategy string `envconfig:"STRATEGY" default:"jwt oneof(jwt, basic)"`
	Jwt      *Jwt   `envconfig:"JWT"`
}

type Redis struct {
	URL          string        `envconfig:"URL" default:"redis://localhost:6379/0"`
	KeyPrefix    string        `envconfig:"KEY_PREFIX" default:""`
	PoolSize     int           `envconfig:"POOL_SIZE" default:"10"`
	DialTimeout  time.Duration `envconfig:"DIAL_TIMEOUT" default:"5s"`
	ReadTimeout  time.Duration `envconfig:"READ_TIMEOUT" default:"3s"`
	WriteTimeout time.Duration `envconfig:"WRITE_TIMEOUT" default:"3s"`
}

type RateLimit struct {
	MaxRequests int           `envconfig:"MAX_REQUESTS" default:"100"`
	Window      time.Duration `envconfig:"WINDOW" default:"1m"`
}

//revive:disable
type Stripe struct {
	Env                  string `envconfig:"ENV" default:"test oneof(test, development, production)"`
	ApiKey               string `envconfig:"API_KEY"`
	SigningSecret        string `envconfig:"SIGNING_SECRET"`
	SuccessPath          string `envconfig:"SUCCESS_PATH" default:"http://localhost:3000/payment/stripe/success/"`
	CancelPath           string `envconfig:"CANCEL_PATH" default:"http://localhost:3000/payment/stripe/cancel/"`
	OnboardingReturnURL  string `envconfig:"ONBOARDING_RETURN_URL" default:"http://localhost:3000/onboarding/return"`
	OnboardingRefreshURL string `envconfig:"ONBOARDING_REFRESH_URL" default:"http://localhost:3000/onboarding/refresh"`
	SkipTLSVerify        bool   `envconfig:"SKIP_TLS_VERIFY" default:"false"` // Skip TLS verification for development
}

//revive:enable
type PaymentProviders struct {
	Stripe *Stripe `envconfig:"STRIPE"`
}

type ExchangeRateApi struct {
	ApiKey      string        `envconfig:"API_KEY"`
	ApiUrl      string        `envconfig:"API_URL" default:""`
	HTTPTimeout time.Duration `envconfig:"HTTP_TIMEOUT" default:"10s"`
}

type ExchangeRateProviders struct {
	ExchangeRateApi *ExchangeRateApi `envconfig:"EXCHANGERATE"`
}

type ExchangeRateCache struct {
	TTL               time.Duration `envconfig:"TTL" default:"15m"`
	MaxRetries        int           `envconfig:"MAX_RETRIES" default:"3"`
	RequestsPerMinute int           `envconfig:"REQUESTS_PER_MINUTE" default:"60"`
	BurstSize         int           `envconfig:"BURST_SIZE" default:"10"`
	EnableFallback    bool          `envconfig:"ENABLE_FALLBACK" default:"true"`
	FallbackTTL       time.Duration `envconfig:"FALLBACK_TTL" default:"1h"`
	Prefix            string        `envconfig:"CACHE_PREFIX" default:"exr:rate:"`
	Url               string        `envconfig:"URL"`
}

type Fee struct {
	ServiceFeePercentage float64 `envconfig:"SERVICE_FEE_PERCENTAGE" default:"0.01"`
}

type Log struct {
	Level      int    `envconfig:"LEVEL" default:"0"`
	Format     string `envconfig:"FORMAT" default:"json"`
	TimeFormat string `envconfig:"TIME_FORMAT" default:"2006-01-02 15:04:05"`
	Prefix     string `envconfig:"PREFIX" default:"[fintech]"`
}

type Server struct {
	Scheme string `envconfig:"SCHEME" default:"http"`
	Host   string `envconfig:"HOST" default:"localhost"`
	Port   int    `envconfig:"PORT" default:"3000"`
}

type App struct {
	Env                      string                 `envconfig:"APP_ENV" default:"development"`
	Server                   *Server                `envconfig:"SERVER"`
	Log                      *Log                   `envconfig:"LOG"`
	DB                       *DB                    `envconfig:"DATABASE"`
	Auth                     *Auth                  `envconfig:"AUTH"`
	ExchangeRateCache        *ExchangeRateCache     `envconfig:"EXCHANGE_RATE_CACHE"`
	ExchangeRateAPIProviders *ExchangeRateProviders `envconfig:"EXCHANGE_RATE_PROVIDER"`
	Redis                    *Redis                 `envconfig:"REDIS"`
	RateLimit                *RateLimit             `envconfig:"RATE_LIMIT"`
	PaymentProviders         *PaymentProviders      `envconfig:"PAYMENT_PROVIDER"`
	Fee                      *Fee                   `envconfig:"FEE"`
}
