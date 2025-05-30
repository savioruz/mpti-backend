package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/savioruz/goth/internal/delivery/http/response"
	"github.com/savioruz/goth/pkg/constant"
	"github.com/savioruz/goth/pkg/failure"
)

func CheckRole(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals(constant.JwtFieldLevel).(string)
		if !ok {
			err := failure.Unauthorized("role information not found")

			return response.WithError(c, err)
		}

		for _, allowedRole := range allowedRoles {
			if role == allowedRole {
				return c.Next()
			}
		}

		err := failure.Forbidden("insufficient permissions")

		return response.WithError(c, err)
	}
}

// AdminOnly protects routes with JWT and Role check for admin role.
func AdminOnly() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := Jwt()(c); err != nil {
			return err
		}

		return CheckRole(constant.UserRoleAdmin)(c)
	}
}
