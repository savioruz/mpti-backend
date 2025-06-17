package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/savioruz/goth/internal/delivery/http/response"
	"github.com/savioruz/goth/internal/domains/payments/dto"
	"github.com/savioruz/goth/internal/domains/payments/service"
	"github.com/savioruz/goth/pkg/constant"
	"github.com/savioruz/goth/pkg/failure"
	"github.com/savioruz/goth/pkg/logger"
)

type Handler struct {
	service   service.PaymentService
	logger    logger.Interface
	validator *validator.Validate
}

func New(s service.PaymentService, l logger.Interface, v *validator.Validate) *Handler {
	return &Handler{
		service:   s,
		logger:    l,
		validator: v,
	}
}

const (
	identifier = "http - payments - %s"

	routepath = "/payments"
)

func (h *Handler) RegisterRoutes(r fiber.Router) {
	payments := r.Group(routepath)

	payments.Post("/callbacks", h.Callbacks)
}

// Callbacks godoc
// @Summary Payment callbacks
// @Description Handle payment callbacks
// @Tags payments
// @Accept json
// @Produce json
// @Param callback body dto.PaymentCallbackRequest true "Payment callback request"
// @Success 200 {object} response.Message
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /payments/callbacks [post]
func (h *Handler) Callbacks(ctx *fiber.Ctx) error {
	var req dto.PaymentCallbackRequest

	if err := ctx.BodyParser(&req); err != nil {
		h.logger.Error(identifier, " - Callbacks - body parser error: %v", err)

		return response.WithError(ctx, err)
	}

	token := ctx.Get(constant.RequestHeaderCallback)

	if err := h.validator.Struct(req); err != nil {
		validationErr := err.Error()
		transformErr := failure.BadRequestFromString(validationErr)

		h.logger.Error(identifier, " - Callbacks - validation error: %v", transformErr)

		return response.WithError(ctx, transformErr)
	}

	if err := h.service.Callbacks(ctx.Context(), req, token); err != nil {
		h.logger.Error(identifier, " - Callbacks - service error: %v", err)

		return response.WithError(ctx, err)
	}

	return response.WithMessage(ctx, fiber.StatusOK, "payment callback processed successfully")
}
