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

	bookingHandler "github.com/savioruz/goth/internal/domains/bookings/handler"
	bookingRepository "github.com/savioruz/goth/internal/domains/bookings/repository"
	bookingService "github.com/savioruz/goth/internal/domains/bookings/service"

	paymentHandler "github.com/savioruz/goth/internal/domains/payments/handler"
	paymentRepository "github.com/savioruz/goth/internal/domains/payments/repository"
	paymentService "github.com/savioruz/goth/internal/domains/payments/service"

	"github.com/savioruz/goth/pkg/httpserver"
	"github.com/savioruz/goth/pkg/jwt"
	"github.com/savioruz/goth/pkg/logger"
	"github.com/savioruz/goth/pkg/mail"
	"github.com/savioruz/goth/pkg/oauth"
	"github.com/savioruz/goth/pkg/postgres"
	"github.com/savioruz/goth/pkg/redis"
	"github.com/savioruz/goth/pkg/supabase"
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

func provideLocationQuerier() locationRepository.Querier {
	return locationRepository.New()
}

var locationDomain = wire.NewSet(
	provideLocationQuerier,
	locationService.New,
	locationHandler.New,
)

func provideFieldQuerier() fieldRepository.Querier {
	return fieldRepository.New()
}

var fieldDomain = wire.NewSet(
	provideFieldQuerier,
	fieldService.New,
	fieldHandler.New,
)

func provideBookingQuerier() bookingRepository.Querier {
	return bookingRepository.New()
}

var bookingDomain = wire.NewSet(
	provideBookingQuerier,
	bookingService.New,
	bookingHandler.New,
)

func providePaymentQuerier() paymentRepository.Querier {
	return paymentRepository.New()
}

var paymentDomain = wire.NewSet(
	providePaymentQuerier,
	paymentService.New,
	paymentHandler.New,
)

var domains = wire.NewSet(
	userDomain,
	authDomain,
	oauthDomain,
	locationDomain,
	fieldDomain,
	bookingDomain,
	paymentDomain,
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
		provideSupabaseClient,
		provideMailService,

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

func provideSupabaseClient(cfg *config.Config, l logger.Interface) (*supabase.Client, error) {
	return supabase.NewClient(supabase.Config{
		AccessKeyID:     cfg.Supabase.AccessKeyID,
		SecretAccessKey: cfg.Supabase.SecretAccessKey,
		EndpointURL:     cfg.Supabase.EndpointURL,
		Region:          cfg.Supabase.Region,
		BucketName:      cfg.Supabase.BucketName,
	})
}

func provideMailService(cfg *config.Config) mail.Service {
	return mail.New(mail.Config{
		SMTPHost:     cfg.Mail.SMTPHost,
		SMTPPort:     cfg.Mail.SMTPPort,
		SMTPUsername: cfg.Mail.SMTPUsername,
		SMTPPassword: cfg.Mail.SMTPPassword,
		FromEmail:    cfg.Mail.FromEmail,
		FromName:     cfg.Mail.FromName,
		TemplatePath: "template", // Default template path from project root
	})
}

func provideHTTPServer(cfg *config.Config, app *fiber.App) *httpserver.Server {
	return httpserver.New(
		httpserver.Port(cfg.HTTP.Port),
		httpserver.App(app),
	)
}
