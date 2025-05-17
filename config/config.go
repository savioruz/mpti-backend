package config

import (
	"fmt"
	"github.com/caarlos0/env/v11"
)

type (
	Config struct {
		App     App
		Cache   Cache
		HTTP    HTTP
		Log     Log
		Pg      Pg
		Redis   Redis
		Swagger Swagger
		JWT     JWT
		OAuth   OAuth
	}

	App struct {
		Name    string `env:"APP_NAME,required"`
		Version string `env:"APP_VERSION,required"`
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
	}
)

func New() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse config failed: %w", err)
	}

	return cfg, nil
}
