package handler

import (
	"errors"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/savioruz/goth/config"
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
	cfg       *config.Config
}

func New(s service.OAuthService, l logger.Interface, v *validator.Validate, cfg *config.Config) *Handler {
	return &Handler{
		service:   s,
		logger:    l,
		validator: v,
		cfg:       cfg,
	}
}

func (h *Handler) RegisterRoutes(r fiber.Router) {
	auth := r.Group("/oauth")

	auth.Get("/google/login", h.GoogleLogin)
	auth.Get("/google/callback", h.GoogleCallback)
}

// GoogleLogin godoc
// @Summary Get Google login URL
// @Description Returns Google OAuth authorization URL and state parameter for the frontend
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} response.Data[dto.OauthGetURLResponse]
// @Failure 500 {object} response.Error
// @Router /oauth/google/login [get]
func (h *Handler) GoogleLogin(ctx *fiber.Ctx) error {
	res, err := h.service.GetGoogleAuthURL()
	if err != nil {
		h.logger.Error("http - v1 - auth - google login - " + err.Error())

		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusOK, res)
}

// GoogleCallback godoc
// @Summary Google OAuth callback
// @Description Handle the Google OAuth callback and redirect to frontend with JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param code query string true "Authorization code from Google"
// @Param state query string true "State parameter for CSRF protection"
// @Param redirect_uri query string false "Frontend URI to redirect to with tokens"
// @Success 302 {string} string "Redirect to frontend with tokens"
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /oauth/google/callback [get]
func (h *Handler) GoogleCallback(ctx *fiber.Ctx) error {
	code := ctx.Query("code")
	if code == "" {
		h.logger.Error("http - v1 - auth - google callback - code is empty")

		return response.WithError(ctx, ErrGoogleLoginCode)
	}

	state := ctx.Query("state")
	if state == "" {
		h.logger.Error("http - v1 - auth - google callback - state is empty")

		return response.WithError(ctx, errors.New("oauth: state parameter is required")) //nolint:err113
	}

	// @TODO: Validate state parameter here if needed

	redirectURI := h.cfg.OAuth.Google.FrontendURL
	if redirectURI == "" {
		h.logger.Error("http - v1 - auth - google callback - redirect URI is not set")

		err := errors.New("redirect URI is not set in configuration") //nolint:err113

		return response.WithError(ctx, err)
	}

	data, err := h.service.HandleGoogleCallback(ctx.Context(), code)
	if err != nil {
		reqID := "unknown"
		if id, ok := ctx.Locals("request_id").(string); ok {
			reqID = id
		}

		h.logger.Error("http - v1 - auth - google callback - request_id: " + reqID + " - " + err.Error())

		return ctx.Redirect(redirectURI + "?error=" + err.Error())
	}

	path := "/oauth/callback"
	redirectURL := fmt.Sprintf("%s%s?access_token=%s&refresh_token=%s&state=%s",
		redirectURI, path, data.AccessToken, data.RefreshToken, state)

	return ctx.Redirect(redirectURL)
}
