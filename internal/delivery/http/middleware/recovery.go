package middleware

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/savioruz/goth/pkg/logger"
)

func buildPanicMessage(ctx *fiber.Ctx, err interface{}) string {
	var result strings.Builder

	result.WriteString(ctx.IP())
	result.WriteString(" - ")
	result.WriteString(ctx.Method())
	result.WriteString(" ")
	result.WriteString(ctx.OriginalURL())
	result.WriteString(" PANIC DETECTED: ")
	result.WriteString(fmt.Sprintf("%v\n%s\n", err, debug.Stack()))

	return result.String()
}

func logPanic(logger logger.Interface) func(c *fiber.Ctx, err interface{}) {
	return func(ctx *fiber.Ctx, err interface{}) {
		logger.Error(buildPanicMessage(ctx, err))
	}
}

func Recovery(l logger.Interface) func(c *fiber.Ctx) error {
	return recover.New(recover.Config{
		EnableStackTrace:  true,
		StackTraceHandler: logPanic(l),
	})
}
