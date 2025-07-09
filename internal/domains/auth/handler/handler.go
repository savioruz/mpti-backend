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
	auth.Get("/verify-email", h.VerifyEmail) // GET with query parameter
	auth.Post("/forgot-password", h.ForgotPassword)
	auth.Get("/reset-password", h.ValidateResetToken) // GET to validate reset token
	auth.Post("/reset-password", h.ResetPassword)     // POST to actually reset password
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

// VerifyEmail godoc
// @Summary Verify email address
// @Description Verify user's email address using verification token
// @Tags auth
// @Accept json
// @Produce json
// @Param token query string true "Email verification token"
// @Success 200 {object} response.Data[dto.EmailVerificationResponse]
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /auth/verify-email [get]
func (h *Handler) VerifyEmail(ctx *fiber.Ctx) error {
	var req dto.EmailVerificationRequest
	if err := ctx.QueryParser(&req); err != nil {
		h.logger.Error("http - auth - verify-email - query parsing error: " + err.Error())

		return response.WithError(ctx, err)
	}

	if err := h.validator.Struct(req); err != nil {
		h.logger.Error("http - auth - verify-email - validate error: " + err.Error())

		return response.WithError(ctx, err)
	}

	data, err := h.service.VerifyEmail(ctx.UserContext(), req)
	if err != nil {
		reqID := "unknown"
		if id, ok := ctx.Locals("request_id").(string); ok {
			reqID = id
		}

		h.logger.Error("http - auth - verify-email - request_id: " + reqID + " - " + err.Error())

		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusOK, data)
}

// ForgotPassword godoc
// @Summary Request password reset
// @Description Send password reset email to user
// @Tags auth
// @Accept json
// @Produce json
// @Param forgot body dto.ForgotPasswordRequest true "Forgot password request"
// @Success 200 {object} response.Data[dto.ForgotPasswordResponse]
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /auth/forgot-password [post]
func (h *Handler) ForgotPassword(ctx *fiber.Ctx) error {
	var req dto.ForgotPasswordRequest
	if err := ctx.BodyParser(&req); err != nil {
		h.logger.Error("http - auth - forgot-password - body parsing error: " + err.Error())

		return response.WithError(ctx, err)
	}

	if err := h.validator.Struct(req); err != nil {
		h.logger.Error("http - auth - forgot-password - validate error: " + err.Error())

		return response.WithError(ctx, err)
	}

	data, err := h.service.ForgotPassword(ctx.UserContext(), req)
	if err != nil {
		reqID := "unknown"
		if id, ok := ctx.Locals("request_id").(string); ok {
			reqID = id
		}

		h.logger.Error("http - auth - forgot-password - request_id: " + reqID + " - " + err.Error())

		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusOK, data)
}

// ResetPassword godoc
// @Summary Reset user password
// @Description Reset user password using reset token
// @Tags auth
// @Accept json
// @Produce json
// @Param reset body dto.ResetPasswordRequest true "Reset password request"
// @Success 200 {object} response.Data[dto.ResetPasswordResponse]
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /auth/reset-password [post]
func (h *Handler) ResetPassword(ctx *fiber.Ctx) error {
	var req dto.ResetPasswordRequest
	if err := ctx.BodyParser(&req); err != nil {
		h.logger.Error("http - auth - reset-password - body parsing error: " + err.Error())

		return response.WithError(ctx, err)
	}

	if err := h.validator.Struct(req); err != nil {
		h.logger.Error("http - auth - reset-password - validate error: " + err.Error())

		return response.WithError(ctx, err)
	}

	data, err := h.service.ResetPassword(ctx.UserContext(), req)
	if err != nil {
		reqID := "unknown"
		if id, ok := ctx.Locals("request_id").(string); ok {
			reqID = id
		}

		h.logger.Error("http - auth - reset-password - request_id: " + reqID + " - " + err.Error())

		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusOK, data)
}

// ValidateResetToken godoc
// @Summary Validate password reset token
// @Description Validate if a password reset token is valid and not expired
// @Tags auth
// @Accept json
// @Produce json
// @Param token query string true "Password reset token"
// @Success 200 {object} response.Data[dto.ValidateResetTokenResponse]
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /auth/reset-password [get]
func (h *Handler) ValidateResetToken(ctx *fiber.Ctx) error {
	var req dto.ValidateResetTokenRequest
	if err := ctx.QueryParser(&req); err != nil {
		h.logger.Error("http - auth - validate-reset-token - query parsing error: " + err.Error())

		return response.WithError(ctx, err)
	}

	if err := h.validator.Struct(req); err != nil {
		h.logger.Error("http - auth - validate-reset-token - validate error: " + err.Error())

		return response.WithError(ctx, err)
	}

	data, err := h.service.ValidateResetToken(ctx.UserContext(), req)
	if err != nil {
		reqID := "unknown"
		if id, ok := ctx.Locals("request_id").(string); ok {
			reqID = id
		}

		h.logger.Error("http - auth - validate-reset-token - request_id: " + reqID + " - " + err.Error())

		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusOK, data)
}
