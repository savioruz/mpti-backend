package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type (
	Config struct {
		App      App
		CORS     CORS
		Cache    Cache
		HTTP     HTTP
		Log      Log
		Pg       Pg
		Redis    Redis
		Swagger  Swagger
		Schedule Schedule
		JWT      JWT
		OAuth    OAuth
		Xendit   Xendit
		Supabase Supabase
		Mail     Mail
	}

	App struct {
		Name     string `env:"APP_NAME,required"`
		Version  string `env:"APP_VERSION,required"`
		Timezone string `env:"APP_TIMEZONE" envDefault:"UTC+7"`
	}

	CORS struct {
		AllowCredentials bool   `env:"APP_CORS_ALLOW_CREDENTIALS"`
		AllowedHeaders   string `env:"APP_CORS_ALLOWED_HEADERS"`
		AllowedMethods   string `env:"APP_CORS_ALLOWED_METHODS"`
		AllowedOrigins   string `env:"APP_CORS_ALLOWED_ORIGINS"`
		Enable           bool   `env:"APP_CORS_ENABLE"`
		MaxAgeSeconds    int    `env:"APP_CORS_MAX_AGE_SECONDS"`
	}

	Cache struct {
		Duration int `env:"CACHE_DURATIONS,required"`
	}

	HTTP struct {
		Port string `env:"HTTP_PORT,required"`
	}

	Log struct {
		Level string `env:"LOG_LEVEL,required" envDefault:"info"`
	}

	Pg struct {
		PoolMax  int    `env:"PG_POOL_MAX,required"`
		Host     string `env:"PG_HOST,required"`
		Port     int    `env:"PG_PORT,required"`
		User     string `env:"PG_USER"`
		Password string `env:"PG_PASSWORD"`
		Dbname   string `env:"PG_DATABASE,required"`
		SSLMode  string `env:"PG_SSLMODE,required"`
		Timezone string `env:"PG_TIMEZONE,required"`
	}

	Redis struct {
		Host     string `env:"REDIS_HOST,required"`
		Port     int    `env:"REDIS_PORT,required"`
		Password string `env:"REDIS_PASSWORD"`
		DB       int    `env:"REDIS_DB"`
	}

	Swagger struct {
		Enabled bool `env:"SWAGGER_ENABLED" envDefault:"false"`
	}

	Schedule struct {
		BookingsExpiration string `env:"SCHEDULE_BOOKINGS_EXPIRATION,required"`
	}

	JWT struct {
		Secret             string `env:"JWT_SECRET,required"`
		AccessTokenExpiry  string `env:"JWT_ACCESS_TOKEN_EXPIRY"  envDefault:"24h"`
		RefreshTokenExpiry string `env:"JWT_REFRESH_TOKEN_EXPIRY" envDefault:"7d"`
	}

	OAuth struct {
		Google GoogleOAuth `env:"OAUTH_GOOGLE"`
	}

	GoogleOAuth struct {
		ClientID     string `env:"OAUTH_GOOGLE_CLIENT_ID,required"`
		ClientSecret string `env:"OAUTH_GOOGLE_CLIENT_SECRET,required"`
		RedirectURL  string `env:"OAUTH_GOOGLE_REDIRECT_URL,required"`
		FrontendURL  string `env:"OAUTH_GOOGLE_FRONTEND_URL,required"`
	}

	Xendit struct {
		APIKey        string `env:"XENDIT_API_KEY,required"`
		CallbackToken string `env:"XENDIT_CALLBACK_TOKEN,required"`
		SuccessURL    string `env:"XENDIT_SUCCESS_URL,required"`
		FailureURL    string `env:"XENDIT_FAILURE_URL,required"`
	}

	Supabase struct {
		AccessKeyID     string `env:"SUPABASE_AWS_ACCESS_KEY_ID,required"`
		SecretAccessKey string `env:"SUPABASE_AWS_SECRET_ACCESS_KEY,required"`
		EndpointURL     string `env:"SUPABASE_ENDPOINT_URL,required"`
		Region          string `env:"SUPABASE_REGION,required"`
		BucketName      string `env:"SUPABASE_BUCKET_NAME,required"`
	}

	Mail struct {
		SMTPHost     string `env:"MAIL_SMTP_HOST,required"`
		SMTPPort     int    `env:"MAIL_SMTP_PORT,required"`
		SMTPUsername string `env:"MAIL_SMTP_USERNAME,required"`
		SMTPPassword string `env:"MAIL_SMTP_PASSWORD,required"`
		FromEmail    string `env:"MAIL_FROM_EMAIL,required"`
		FromName     string `env:"MAIL_FROM_NAME,required"`
	}
)

func New() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse config failed: %w", err)
	}

	return cfg, nil
}
