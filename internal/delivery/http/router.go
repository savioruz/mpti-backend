package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"github.com/savioruz/goth/config"
	_ "github.com/savioruz/goth/docs" // Swagger docs
	authHandler "github.com/savioruz/goth/internal/domains/auth/handler"
	oauthHandler "github.com/savioruz/goth/internal/domains/oauth/handler"
	userHandler "github.com/savioruz/goth/internal/domains/user/handler"

	"github.com/savioruz/goth/internal/delivery/http/middleware"
	"github.com/savioruz/goth/pkg/logger"
)

// NewRouter initializes the HTTP router and registers the routes for the application.
// Swagger spec:
// @title Goth API
// @BasePath /v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func NewRouter(
	app *fiber.App,
	cfg *config.Config,
	l logger.Interface,
	authHandler *authHandler.Handler,
	oauthHandler *oauthHandler.Handler,
	userHandler *userHandler.Handler,
) {
	// Options
	app.Use(middleware.Logger(l))
	app.Use(middleware.Recovery(l))
	app.Use(middleware.RequestID())

	if cfg.Swagger.Enabled {
		app.Get("/swagger/*", swagger.HandlerDefault)
	}

	apiV1Group := app.Group("/v1")
	{
		authHandler.RegisterRoutes(apiV1Group)
		oauthHandler.RegisterRoutes(apiV1Group)
		userHandler.RegisterRoutes(apiV1Group)
	}

	app.Use("*", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "route not found",
		})
	})
}
