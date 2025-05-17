package middleware

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/savioruz/goth/pkg/logger"
)

func buildRequestMessage(ctx *fiber.Ctx) string {
	var result strings.Builder

	result.WriteString(ctx.IP())
	result.WriteString(" - ")
	result.WriteString(ctx.Method())
	result.WriteString(" ")
	result.WriteString(ctx.OriginalURL())
	result.WriteString(" - ")
	result.WriteString(strconv.Itoa(ctx.Response().StatusCode()))
	result.WriteString(" ")
	result.WriteString(strconv.Itoa(len(ctx.Response().Body())))

	return result.String()
}

func Logger(l logger.Interface) func(c *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		start := time.Now()

		err := ctx.Next()

		duration := time.Since(start).Milliseconds()

		l.Info(buildRequestMessage(ctx) + " - " + strconv.FormatInt(duration, 10) + "ms")

		return err
	}
}
