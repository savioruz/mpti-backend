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
	payments.Get("/", h.GetPayments)
	payments.Get("/booking/:booking_id", h.GetPaymentsByBookingID)
}

// Callbacks godoc
// @Summary Payment callbacks
// @Description Handle payment callbacks
// @Tags payments
// @Accept json
// @Produce json
// @Param callback body dto.CallbackPaymentInvoice true "Payment callback request"
// @Success 200 {object} response.Message
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /payments/callbacks [post]
func (h *Handler) Callbacks(ctx *fiber.Ctx) error {
	var req dto.CallbackPaymentInvoice

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

// GetPayments godoc
// @Summary Get payments
// @Description Get all payments with optional filtering and pagination
// @Tags payments
// @Accept json
// @Produce json
// @Param payment_method query string false "Filter by payment method"
// @Param payment_status query string false "Filter by payment status"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10, max: 100)"
// @Success 200 {object} response.Data[dto.PaginatedPaymentResponse]
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /payments/ [get]
func (h *Handler) GetPayments(ctx *fiber.Ctx) error {
	var req dto.GetPaymentsRequest

	if err := ctx.QueryParser(&req); err != nil {
		h.logger.Error(identifier, " - GetPayments - query parser error: %v", err)

		return response.WithError(ctx, err)
	}

	payments, err := h.service.GetPayments(ctx.Context(), req)
	if err != nil {
		h.logger.Error(identifier, " - GetPayments - service error: %v", err)

		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusOK, payments)
}

// GetPaymentsByBookingID godoc
// @Summary Get payments by booking ID
// @Description Get all payments for a specific booking
// @Tags payments
// @Accept json
// @Produce json
// @Param booking_id path string true "Booking ID"
// @Success 200 {object} response.Data[[]dto.PaymentResponse]
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /payments/booking/{booking_id} [get]
func (h *Handler) GetPaymentsByBookingID(ctx *fiber.Ctx) error {
	bookingID := ctx.Params("booking_id")

	if bookingID == "" {
		h.logger.Error(identifier, " - GetPaymentsByBookingID - booking ID is required")

		return response.WithError(ctx, failure.BadRequestFromString("booking ID is required"))
	}

	payments, err := h.service.GetPaymentsByBookingID(ctx.Context(), bookingID)
	if err != nil {
		h.logger.Error(identifier, " - GetPaymentsByBookingID - service error: %v", err)

		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusOK, payments)
}
