package handler

import (
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/savioruz/goth/internal/delivery/http/middleware"
	"github.com/savioruz/goth/internal/delivery/http/response"

	// Register the swagger docs
	_ "github.com/savioruz/goth/internal/domains/user/dto"
	"github.com/savioruz/goth/internal/domains/user/service"
	"github.com/savioruz/goth/pkg/logger"
)

var (
	ErrEmailNil       = errors.New("email is nil")
	ErrEmailNotString = errors.New("email is not string")
)

type Handler struct {
	service service.UserService
	logger  logger.Interface
}

func New(s service.UserService, l logger.Interface) *Handler {
	return &Handler{
		service: s,
		logger:  l,
	}
}

func (h *Handler) RegisterRoutes(r fiber.Router) {
	auth := r.Group("/users")

	auth.Get("/profile", middleware.Jwt(), h.Profile)
}

// Profile godoc
// @Summary Get user profile
// @Description Get user profile
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {object} response.Data[dto.UserProfileResponse]
// @Failure 500 {object} response.Error
// @Router /users/profile [get]
// @Security BearerAuth
func (h *Handler) Profile(ctx *fiber.Ctx) error {
	localEmail := ctx.Locals("email")
	if localEmail == nil {
		h.logger.Error("http - user - profile - email is nil")

		return response.WithError(ctx, ErrEmailNil)
	}

	email, ok := localEmail.(string)
	if !ok {
		h.logger.Error("http - user - profile - email is not string")

		return response.WithError(ctx, ErrEmailNotString)
	}

	data, err := h.service.Profile(ctx.UserContext(), email)
	if err != nil {
		reqID := "unknown"
		if id, ok := ctx.Locals("request_id").(string); ok {
			reqID = id
		}

		h.logger.Error("http - user - profile - request_id: " + reqID + " - " + err.Error())

		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusOK, data)
}
