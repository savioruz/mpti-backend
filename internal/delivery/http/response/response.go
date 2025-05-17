package response

import (
	"github.com/gofiber/fiber/v2"
	"github.com/savioruz/goth/pkg/failure"
)

type Data[T any] struct {
	Data T `json:"data,omitempty"`
}

type Error struct {
	Error *string `json:"error,omitempty"`
}

func WithJSON(ctx *fiber.Ctx, code int, payload interface{}) error {
	err := response(ctx, code, Data[any]{Data: payload})
	if err != nil {
		return err
	}

	return nil
}

func WithError(ctx *fiber.Ctx, err error) error {
	code := failure.GetCode(err)
	errMsg := err.Error()

	return response(ctx, code, Error{Error: &errMsg})
}

func response(ctx *fiber.Ctx, code int, payload interface{}) error {
	if payload == nil {
		return ctx.SendStatus(code)
	}

	ctx.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)

	if err := ctx.Status(code).JSON(payload); err != nil {
		return err
	}

	return nil
}
