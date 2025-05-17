package handler

import (
	"errors"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/savioruz/goth/internal/delivery/http/response"
	"github.com/savioruz/goth/internal/domains/oauth/service"

	// Register dto for swagger docs
	_ "github.com/savioruz/goth/internal/domains/user/dto"
	"github.com/savioruz/goth/pkg/logger"
)

var (
	ErrGoogleLoginCode = errors.New("oauth: google login code required")
)

type Handler struct {
	service   service.OAuthService
	logger    logger.Interface
	validator *validator.Validate
}

func New(s service.OAuthService, l logger.Interface, v *validator.Validate) *Handler {
	return &Handler{
		service:   s,
		logger:    l,
		validator: v,
	}
}

func (h *Handler) RegisterRoutes(r fiber.Router) {
	auth := r.Group("/oauth")

	auth.Get("/google/login", h.GoogleLogin)
	auth.Get("/google/callback", h.GoogleCallback)
}

// GoogleLogin godoc
// @Summary Login with Google
// @Description Redirects to Google OAuth consent screen
// @Tags auth
// @Accept json
// @Produce json
// @Success 302 {string} string "Redirect to Google"
// @Failure 500 {object} response.Error
// @Router /oauth/google/login [get]
func (h *Handler) GoogleLogin(ctx *fiber.Ctx) error {
	url := h.service.GetGoogleAuthURL()

	if err := ctx.Redirect(url); err != nil {
		h.logger.Error("http - v1 - auth - google login - redirect error: %w", err)

		return response.WithError(ctx, err)
	}

	return nil
}

// GoogleCallback godoc
// @Summary Google OAuth callback
// @Description Handle the Google OAuth callback and return JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param code query string true "Authorization code from Google"
// @Success 200 {object} response.Data[dto.UserLoginResponse]
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /oauth/google/callback [get]
func (h *Handler) GoogleCallback(ctx *fiber.Ctx) error {
	code := ctx.Query("code")
	if code == "" {
		h.logger.Error("http - v1 - auth - google callback - code is empty")

		return response.WithError(ctx, ErrGoogleLoginCode)
	}

	data, err := h.service.HandleGoogleCallback(ctx.Context(), code)
	if err != nil {
		reqID := "unknown"
		if id, ok := ctx.Locals("request_id").(string); ok {
			reqID = id
		}

		h.logger.Error("http - v1 - auth - google callback - request_id: " + reqID + " - " + err.Error())

		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusOK, data)
}
