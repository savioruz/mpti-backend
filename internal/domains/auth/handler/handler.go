package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/savioruz/goth/internal/delivery/http/response"
	"github.com/savioruz/goth/internal/domains/auth/service"
	"github.com/savioruz/goth/internal/domains/user/dto"
	"github.com/savioruz/goth/pkg/logger"
)

type Handler struct {
	service   service.AuthService
	logger    logger.Interface
	validator *validator.Validate
}

func New(s service.AuthService, l logger.Interface, v *validator.Validate) *Handler {
	return &Handler{
		service:   s,
		logger:    l,
		validator: v,
	}
}

func (h *Handler) RegisterRoutes(r fiber.Router) {
	auth := r.Group("/auth")

	auth.Post("/register", h.Register)
	auth.Post("/login", h.Login)
}

// Register godoc
// @Summary Register new user
// @Description Register new user with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param register body dto.UserRegisterRequest true "User register request"
// @Success 201 {object} response.Data[dto.UserRegisterResponse]
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /auth/register [post]
func (h *Handler) Register(ctx *fiber.Ctx) error {
	var req dto.UserRegisterRequest
	if err := ctx.BodyParser(&req); err != nil {
		h.logger.Error("http - auth - register - body parsing error: " + err.Error())

		return response.WithError(ctx, err)
	}

	if err := h.validator.Struct(req); err != nil {
		h.logger.Error("http - auth - validate error: " + err.Error())

		return response.WithError(ctx, err)
	}

	data, err := h.service.Register(ctx.UserContext(), req)
	if err != nil {
		reqID := "unknown"
		if id, ok := ctx.Locals("request_id").(string); ok {
			reqID = id
		}

		h.logger.Error("http - auth - register - request_id: " + reqID + " - " + err.Error())

		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusCreated, data)
}

// Login godoc
// @Summary Login user
// @Description Login user with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param login body dto.UserLoginRequest true "User login request"
// @Success 201 {object} response.Data[dto.UserLoginResponse]
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /auth/login [post]
func (h *Handler) Login(ctx *fiber.Ctx) error {
	var req dto.UserLoginRequest
	if err := ctx.BodyParser(&req); err != nil {
		h.logger.Error("http - auth - login - body parsing error: " + err.Error())

		return response.WithError(ctx, err)
	}

	if err := h.validator.Struct(req); err != nil {
		h.logger.Error("http - auth - login - validate error: " + err.Error())

		return response.WithError(ctx, err)
	}

	data, err := h.service.Login(ctx.UserContext(), req)
	if err != nil {
		reqID := "unknown"
		if id, ok := ctx.Locals("request_id").(string); ok {
			reqID = id
		}

		h.logger.Error("http - auth - login - request_id: " + reqID + " - " + err.Error())

		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusOK, data)
}
