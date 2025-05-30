//go:build wireinject
// +build wireinject

package app

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/wire"
	"github.com/savioruz/goth/config"
	"github.com/savioruz/goth/internal/delivery/http"

	authHandler "github.com/savioruz/goth/internal/domains/auth/handler"
	authService "github.com/savioruz/goth/internal/domains/auth/service"

	oauthHandler "github.com/savioruz/goth/internal/domains/oauth/handler"
	oauthService "github.com/savioruz/goth/internal/domains/oauth/service"

	locationHandler "github.com/savioruz/goth/internal/domains/locations/handler"
	locationRepository "github.com/savioruz/goth/internal/domains/locations/repository"
	locationService "github.com/savioruz/goth/internal/domains/locations/service"

	userHandler "github.com/savioruz/goth/internal/domains/user/handler"
	userRepository "github.com/savioruz/goth/internal/domains/user/repository"
	userService "github.com/savioruz/goth/internal/domains/user/service"

	fieldHandler "github.com/savioruz/goth/internal/domains/fields/handler"
	fieldRepository "github.com/savioruz/goth/internal/domains/fields/repository"
	fieldService "github.com/savioruz/goth/internal/domains/fields/service"

	"github.com/savioruz/goth/pkg/httpserver"
	"github.com/savioruz/goth/pkg/jwt"
	"github.com/savioruz/goth/pkg/logger"
	"github.com/savioruz/goth/pkg/oauth"
	"github.com/savioruz/goth/pkg/postgres"
	"github.com/savioruz/goth/pkg/redis"
)

// Application represents the dependency-injected app
type Application struct {
	HTTPServer *httpserver.Server
	Logger     logger.Interface
	PG         *postgres.Postgres
	Redis      *redis.Redis
	JWT        *jwt.JWT
}

func provideUserQuerier() userRepository.Querier {
	return userRepository.New()
}

var userDomain = wire.NewSet(
	provideUserQuerier,
	userService.New,
	userHandler.New,
)

var authDomain = wire.NewSet(
	authService.New,
	authHandler.New,
)

var oauthDomain = wire.NewSet(
	oauthService.New,
	oauthHandler.New,
)

var locationDomain = wire.NewSet(
	locationRepository.New,
	locationService.New,
	locationHandler.New,
	wire.Bind(new(locationRepository.Querier), new(*locationRepository.Queries)),
)

var fieldDomain = wire.NewSet(
	fieldRepository.New,
	fieldService.New,
	fieldHandler.New,
	wire.Bind(new(fieldRepository.Querier), new(*fieldRepository.Queries)),
)

var domains = wire.NewSet(
	userDomain,
	authDomain,
	oauthDomain,
	locationDomain,
	fieldDomain,
)

func InitializeApp(cfg *config.Config) (*Application, error) {
	wire.Build(
		// Infrastructure providers
		provideLogger,
		providePostgres,
		providePgxIface,
		provideValidator,
		provideRedis,
		provideRedisCache,
		provideJWT,
		provideGoogleOAuth,

		domains,

		wire.Struct(new(http.Handlers), "*"),

		// HTTP server
		provideRouter,
		provideHTTPServer,

		// Application
		wire.Struct(new(Application), "*"),
	)

	return &Application{}, nil
}

func provideRouter(
	cfg *config.Config,
	l logger.Interface,
	h http.Handlers,
) *fiber.App {
	app := fiber.New()

	http.NewRouter(
		app,
		cfg,
		l,
		h,
	)

	return app
}

func provideLogger(cfg *config.Config) logger.Interface {
	return logger.New(cfg.Log.Level)
}

func provideJWT(cfg *config.Config) *jwt.JWT {
	jwt.Initialize(cfg.App.Name, cfg.JWT.Secret, jwt.ParseDuration(cfg.JWT.AccessTokenExpiry), jwt.ParseDuration(cfg.JWT.RefreshTokenExpiry))
	return jwt.GetInstance()
}

func providePostgres(cfg *config.Config, l logger.Interface) (*postgres.Postgres, error) {
	dsn := postgres.ConnectionBuilder(cfg.Pg.Host, cfg.Pg.Port, cfg.Pg.User, cfg.Pg.Password, cfg.Pg.Dbname, cfg.Pg.SSLMode, cfg.Pg.Timezone)
	pg, err := postgres.New(dsn, postgres.MaxPoolSize(cfg.Pg.PoolMax))
	if err != nil {
		return nil, err
	}
	return pg, nil
}

func providePgxIface(pg *postgres.Postgres) postgres.PgxIface {
	return pg.Pool
}

func provideRedis(cfg *config.Config) (*redis.Redis, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port)
	return redis.New(addr, cfg.Redis.Password, cfg.Redis.DB)
}

func provideRedisCache(r *redis.Redis, l logger.Interface) redis.IRedisCache {
	return redis.NewRedisCache(r.Client, l)
}

func provideValidator() *validator.Validate {
	return validator.New(validator.WithRequiredStructEnabled())
}

func provideGoogleOAuth(cfg *config.Config) oauth.GoogleProviderIface {
	return oauth.NewGoogleProvider(cfg.OAuth.Google.ClientID, cfg.OAuth.Google.ClientSecret, cfg.OAuth.Google.RedirectURL)
}

func provideHTTPServer(cfg *config.Config, app *fiber.App) *httpserver.Server {
	return httpserver.New(
		httpserver.Port(cfg.HTTP.Port),
		httpserver.App(app),
	)
}
