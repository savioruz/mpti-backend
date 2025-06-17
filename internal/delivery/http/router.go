package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"github.com/savioruz/goth/config"
	_ "github.com/savioruz/goth/docs" // Swagger docs
	authHandler "github.com/savioruz/goth/internal/domains/auth/handler"
	bookingHandler "github.com/savioruz/goth/internal/domains/bookings/handler"
	fieldHandler "github.com/savioruz/goth/internal/domains/fields/handler"
	locationHandler "github.com/savioruz/goth/internal/domains/locations/handler"
	oauthHandler "github.com/savioruz/goth/internal/domains/oauth/handler"
	paymentHandler "github.com/savioruz/goth/internal/domains/payments/handler"
	userHandler "github.com/savioruz/goth/internal/domains/user/handler"

	"github.com/savioruz/goth/internal/delivery/http/middleware"
	"github.com/savioruz/goth/pkg/logger"
)

type Handlers struct {
	Auth     *authHandler.Handler
	OAuth    *oauthHandler.Handler
	User     *userHandler.Handler
	Location *locationHandler.Handler
	Field    *fieldHandler.Handler
	Booking  *bookingHandler.Handler
	Payment  *paymentHandler.Handler
}

// NewRouter initializes the HTTP router and registers the routes for the application.
// Swagger spec:
// @title mpti API
// @BasePath /v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func NewRouter(
	app *fiber.App,
	cfg *config.Config,
	l logger.Interface,
	handlers Handlers,
) {
	// Options
	app.Use(middleware.Logger(l))
	app.Use(middleware.Recovery(l))
	app.Use(middleware.RequestID())
	app.Use(middleware.CORS(cfg))

	if cfg.Swagger.Enabled {
		app.Get("/swagger/*", swagger.HandlerDefault)
	}

	apiV1Group := app.Group("/v1")
	{
		handlers.Auth.RegisterRoutes(apiV1Group)
		handlers.OAuth.RegisterRoutes(apiV1Group)
		handlers.User.RegisterRoutes(apiV1Group)
		handlers.Location.RegisterRoutes(apiV1Group)
		handlers.Field.RegisterRoutes(apiV1Group)
		handlers.Booking.RegisterRoutes(apiV1Group)
		handlers.Payment.RegisterRoutes(apiV1Group)
	}

	app.Use("*", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "route not found",
		})
	})
}
