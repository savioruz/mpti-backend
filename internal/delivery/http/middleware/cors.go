package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/savioruz/goth/config"
)

func CORS(cfg *config.Config) fiber.Handler {
	if cfg.CORS.Enable {
		return cors.New(cors.Config{
			AllowOrigins:     cfg.CORS.AllowedOrigins,
			AllowMethods:     cfg.CORS.AllowedMethods,
			AllowHeaders:     cfg.CORS.AllowedHeaders,
			AllowCredentials: cfg.CORS.AllowCredentials,
			MaxAge:           cfg.CORS.MaxAgeSeconds,
		})
	}

	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}
