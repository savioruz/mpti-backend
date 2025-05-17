package middleware

import (
	"github.com/savioruz/goth/pkg/failure"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/savioruz/goth/internal/delivery/http/response"
	"github.com/savioruz/goth/pkg/jwt"
)

func Jwt() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			err := failure.Unauthorized("missing authorization header")

			return response.WithError(c, err)
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			err := failure.Unauthorized("invalid authorization header format")

			return response.WithError(c, err)
		}

		claims, err := jwt.ValidateToken(parts[1])
		if err != nil {
			err := failure.Unauthorized("invalid token")

			return response.WithError(c, err)
		}

		if claims != nil {
			c.Locals("user_id", claims.ID)
			c.Locals("email", claims.Email)
			c.Locals("level", claims.Level)
		}

		return c.Next()
	}
}
