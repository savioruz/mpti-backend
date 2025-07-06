package handler

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/savioruz/goth/internal/delivery/http/middleware"
	"github.com/savioruz/goth/internal/delivery/http/response"
	"github.com/savioruz/goth/internal/domains/bookings/dto"
	"github.com/savioruz/goth/internal/domains/bookings/service"
	"github.com/savioruz/goth/pkg/constant"
	"github.com/savioruz/goth/pkg/failure"
	"github.com/savioruz/goth/pkg/gdto"
	"github.com/savioruz/goth/pkg/logger"
)

type Handler struct {
	service   service.BookingService
	logger    logger.Interface
	validator *validator.Validate
}

func New(s service.BookingService, l logger.Interface, v *validator.Validate) *Handler {
	return &Handler{
		service:   s,
		logger:    l,
		validator: v,
	}
}

const (
	identifier = "http - booking - %s"

	routepath = "/bookings"
)

func (h *Handler) RegisterRoutes(r fiber.Router) {
	bookings := r.Group(routepath)

	bookings.Post("/", middleware.Jwt(), h.CreateBooking)
	bookings.Get("/:id", h.GetBookingByID)
	bookings.Post("/slots", h.GetBookedSlots)
	bookings.Put("/:id/cancel", middleware.Jwt(), h.CancelUserBooking)

	r.Get("/users/bookings", middleware.Jwt(), h.GetUserBookings)
}

// CreateBooking godoc
// @Summary Create new booking
// @Description Create new booking
// @Tags bookings
// @Accept json
// @Produce json
// @Param booking body dto.CreateBookingRequest true "Create booking request"
// @Success 201 {object} response.Data[string]
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /bookings/ [post]
// @Security BearerAuth
func (h *Handler) CreateBooking(ctx *fiber.Ctx) error {
	var req dto.CreateBookingRequest
	if err := ctx.BodyParser(&req); err != nil {
		h.logger.Error(identifier, "error parsing request body: "+err.Error())

		return response.WithError(ctx, err)
	}

	if err := h.validator.Struct(req); err != nil {
		validationErr := err.Error()
		transformErr := failure.BadRequestFromString(validationErr)

		h.logger.Error("http - field - create - validate error: " + validationErr)

		return response.WithError(ctx, transformErr)
	}

	userRaw := ctx.Locals(constant.JwtFieldUser)
	if userRaw == nil {
		h.logger.Error(identifier, "user not found in context")

		return response.WithError(ctx, failure.Unauthorized("user not authenticated"))
	}

	user, ok := userRaw.(string)
	if !ok {
		h.logger.Error(identifier, "invalid user type in context")

		return response.WithError(ctx, constant.ErrInvalidContextUserType)
	}

	emailRaw := ctx.Locals(constant.JwtFieldEmail)
	if emailRaw == nil {
		h.logger.Error(identifier, "email not found in context")

		return response.WithError(ctx, failure.Unauthorized("email not authenticated"))
	}

	email, ok := emailRaw.(string)
	if !ok {
		h.logger.Error(identifier, "invalid email type in context")

		return response.WithError(ctx, constant.ErrInvalidContextUserType)
	}

	res, err := h.service.CreateBooking(ctx.Context(), req, user, email)
	if err != nil {
		h.logger.Error(identifier, "error creating booking: "+err.Error())

		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusCreated, res)
}

// GetBookingByID godoc
// @Summary Get booking by ID
// @Description Get booking by ID
// @Tags bookings
// @Accept json
// @Produce json
// @Param id path string true "Booking ID"
// @Success 200 {object} response.Data[dto.BookingResponse]
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /bookings/{id} [get]
// @Security BearerAuth
func (h *Handler) GetBookingByID(ctx *fiber.Ctx) error {
	id := ctx.Params(constant.RequestParamID)
	if err := h.validator.Var(id, constant.RequestValidateUUID); err != nil {
		err = failure.BadRequestFromString("invalid booking id format")

		h.logger.Error(identifier, "get - validate error: %w", err)

		return response.WithError(ctx, err)
	}

	res, err := h.service.GetBookingByID(ctx.Context(), id)
	if err != nil {
		h.logger.Error(identifier, "error getting booking by id: %w", err)

		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusOK, res)
}

// GetUserBookings godoc
// @Summary Get user bookings
// @Description Get bookings for the authenticated user
// @Tags bookings
// @Accept json
// @Produce json
// @Param pagination query gdto.PaginationRequest false "Pagination parameters filter by status"
// @Success 200 {object} response.Data[dto.GetBookingsResponse]
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /users/bookings [get]
func (h *Handler) GetUserBookings(ctx *fiber.Ctx) error {
	userRaw := ctx.Locals(constant.JwtFieldUser)
	if userRaw == nil {
		h.logger.Error(identifier, "user not found in context")

		return response.WithError(ctx, failure.Unauthorized("user not authenticated"))
	}

	user, ok := userRaw.(string)
	if !ok {
		h.logger.Error(identifier, "invalid user type in context")

		return response.WithError(ctx, constant.ErrInvalidContextUserType)
	}

	var req gdto.PaginationRequest
	if err := ctx.QueryParser(&req); err != nil {
		h.logger.Error(identifier, "error parsing query parameters: "+err.Error())

		return response.WithError(ctx, err)
	}

	res, err := h.service.GetUserBookings(ctx.Context(), user, req)
	if err != nil {
		h.logger.Error(identifier, "error getting user bookings: "+err.Error())

		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusOK, res)
}

// GetBookedSlots godoc
// @Summary Get booked slots
// @Description Get booked slots for a specific date and field
// @Tags bookings
// @Accept json
// @Produce json
// @Param request body dto.GetBookedSlotsRequest true "Get booked slots request"
// @Success 200 {object} response.Data[dto.GetBookedSlotsResponse]
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /bookings/slots [post]
func (h *Handler) GetBookedSlots(ctx *fiber.Ctx) error {
	var req dto.GetBookedSlotsRequest
	if err := ctx.BodyParser(&req); err != nil {
		h.logger.Error(identifier, "error parsing request body: "+err.Error())

		return response.WithError(ctx, err)
	}

	if err := h.validator.Struct(req); err != nil {
		validationErr := err.Error()
		transformErr := failure.BadRequestFromString(validationErr)

		h.logger.Error(identifier, "get booked slots - validate error: "+validationErr)

		return response.WithError(ctx, transformErr)
	}

	res, err := h.service.GetBookedSlots(ctx.Context(), req)
	if err != nil {
		h.logger.Error(identifier, "error getting booked slots: "+err.Error())

		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusOK, res)
}

// CancelUserBooking godoc
// @Summary Cancel user booking
// @Description Cancel a booking for the authenticated user
// @Tags bookings
// @Accept json
// @Produce json
// @Param id path string true "Booking ID"
// @Success 200 {object} response.Data[string]
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /bookings/{id}/cancel [put]
// @Security BearerAuth
func (h *Handler) CancelUserBooking(ctx *fiber.Ctx) error {
	id := ctx.Params(constant.RequestParamID)
	if err := h.validator.Var(id, constant.RequestValidateUUID); err != nil {
		err = failure.BadRequestFromString("invalid booking id format")

		h.logger.Error(identifier, "cancel - validate error: %w", err)

		return response.WithError(ctx, err)
	}

	userRaw := ctx.Locals(constant.JwtFieldUser)
	if userRaw == nil {
		h.logger.Error(identifier, "user not found in context")

		return response.WithError(ctx, failure.Unauthorized("user not authenticated"))
	}

	user, ok := userRaw.(string)
	if !ok {
		h.logger.Error(identifier, "invalid user type in context")

		return response.WithError(ctx, constant.ErrInvalidContextUserType)
	}

	req := dto.CancelUserBookingRequest{
		BookingID: id,
		UserID:    user,
	}

	err := h.service.CancelUserBooking(ctx.Context(), req)
	if err != nil {
		h.logger.Error(identifier, "error canceling booking: %w", err)

		return response.WithError(ctx, err)
	}

	res := fmt.Sprintf("Booking %s cancelled", id)

	return response.WithMessage(ctx, fiber.StatusOK, res)
}
