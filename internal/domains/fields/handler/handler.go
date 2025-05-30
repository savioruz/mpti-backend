package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/savioruz/goth/internal/delivery/http/middleware"
	"github.com/savioruz/goth/internal/delivery/http/response"
	"github.com/savioruz/goth/internal/domains/fields/dto"
	"github.com/savioruz/goth/internal/domains/fields/service"
	"github.com/savioruz/goth/pkg/failure"
	"github.com/savioruz/goth/pkg/gdto"
	"github.com/savioruz/goth/pkg/logger"
)

type Handler struct {
	service   service.FieldService
	logger    logger.Interface
	validator *validator.Validate
}

func New(s service.FieldService, l logger.Interface, v *validator.Validate) *Handler {
	return &Handler{
		service:   s,
		logger:    l,
		validator: v,
	}
}

const (
	identifier = "http - field - %s"
)

func (h *Handler) RegisterRoutes(r fiber.Router) {
	field := r.Group("/fields")

	field.Post("/", middleware.AdminOnly(), h.Create)
	field.Get("/:id", h.Get)
	field.Get("/", h.GetAll)
	field.Patch("/:id", middleware.AdminOnly(), h.Update)
	field.Delete("/:id", middleware.AdminOnly(), h.Delete)

	r.Group("/locations").Get("/:location_id/fields", h.GetByLocationID)
}

// Create Field godoc
// @Summary Create new field
// @Description Create new field
// @Tags fields
// @Accept json
// @Produce json
// @Param field body dto.FieldCreateRequest true "Field create request"
// @Success 201 {object} response.Data[string]
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /fields/ [post]
// @Security BearerAuth
func (h *Handler) Create(ctx *fiber.Ctx) error {
	var req dto.FieldCreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		h.logger.Error("http - field - create - body parsing error: " + err.Error())

		return response.WithError(ctx, err)
	}

	if err := h.validator.Struct(req); err != nil {
		validationErr := err.Error()
		transformErr := failure.BadRequestFromString(validationErr)

		h.logger.Error("http - field - create - validate error: " + validationErr)

		return response.WithError(ctx, transformErr)
	}

	data, err := h.service.Create(ctx.UserContext(), req)
	if err != nil {
		reqID := "unknown"
		if id, ok := ctx.Locals("request_id").(string); ok {
			reqID = id
		}

		h.logger.Error("http - field - create - request_id: " + reqID + " - " + err.Error())

		return response.WithError(ctx, err)
	}

	return response.WithMessage(ctx, fiber.StatusCreated, data)
}

// Get Field godoc
// @Summary Get field by ID
// @Description Get field by ID
// @Tags fields
// @Accept json
// @Produce json
// @Param id path string true "Field ID"
// @Success 200 {object} response.Data[dto.FieldResponse]
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /fields/{id} [get]
func (h *Handler) Get(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		h.logger.Error("http - field - get - id is empty")

		return response.WithError(ctx, failure.BadRequestFromString("id is required"))
	}

	data, err := h.service.Get(ctx.UserContext(), id)
	if err != nil {
		reqID := "unknown"
		if id, ok := ctx.Locals("request_id").(string); ok {
			reqID = id
		}

		h.logger.Error("http - field - get - request_id: " + reqID + " - " + err.Error())

		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusOK, data)
}

// GetAll Field godoc
// @Summary Get all fields
// @Description Get all fields
// @Tags fields
// @Accept json
// @Produce json
// @Param pagination query gdto.PaginationRequest false "Pagination request"
// @Success 200 {object} response.Data[dto.GetFieldsResponse]
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /fields/ [get]
func (h *Handler) GetAll(ctx *fiber.Ctx) error {
	var req gdto.PaginationRequest
	if err := ctx.QueryParser(&req); err != nil {
		h.logger.Error("http - field - get all - query parsing error: " + err.Error())

		return response.WithError(ctx, failure.BadRequestFromString(err.Error()))
	}

	if err := h.validator.Struct(req); err != nil {
		h.logger.Error("http - field - get all - validate error: " + err.Error())

		return response.WithError(ctx, failure.BadRequestFromString(err.Error()))
	}

	data, err := h.service.GetAll(ctx.UserContext(), req)
	if err != nil {
		reqID := "unknown"
		if id, ok := ctx.Locals("request_id").(string); ok {
			reqID = id
		}

		h.logger.Error("http - field - get all - request_id: " + reqID + " - " + err.Error())

		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusOK, data)
}

// Update Field godoc
// @Summary Update field by ID
// @Description Update field by ID
// @Tags fields
// @Accept json
// @Produce json
// @Param id path string true "Field ID"
// @Param field body dto.FieldUpdateRequest true "Field update request"
// @Success 200 {object} response.Data[dto.FieldResponse]
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /fields/{id} [patch]
// @Security BearerAuth
func (h *Handler) Update(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		h.logger.Error("http - field - update - id is empty")

		return response.WithError(ctx, failure.BadRequestFromString("id is required"))
	}

	var req dto.FieldUpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		h.logger.Error("http - field - update - body parsing error: " + err.Error())

		return response.WithError(ctx, err)
	}

	if err := h.validator.Struct(req); err != nil {
		validationErr := err.Error()
		transformErr := failure.BadRequestFromString(validationErr)

		h.logger.Error("http - field - update - validate error: " + validationErr)

		return response.WithError(ctx, transformErr)
	}

	data, err := h.service.Update(ctx.UserContext(), id, req)
	if err != nil {
		reqID := "unknown"
		if id, ok := ctx.Locals("request_id").(string); ok {
			reqID = id
		}

		h.logger.Error("http - field - update - request_id: " + reqID + " - " + err.Error())

		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusOK, data)
}

// Delete Field godoc
// @Summary Delete field by ID
// @Description Delete field by ID
// @Tags fields
// @Accept json
// @Produce json
// @Param id path string true "Field ID"
// @Success 200 {object} response.Message
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /fields/{id} [delete]
// @Security BearerAuth
func (h *Handler) Delete(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		h.logger.Error("http - field - delete - id is empty")

		return response.WithError(ctx, failure.BadRequestFromString("id is required"))
	}

	res, err := h.service.Delete(ctx.UserContext(), id)
	if err != nil {
		reqID := "unknown"
		if id, ok := ctx.Locals("request_id").(string); ok {
			reqID = id
		}

		h.logger.Error("http - field - delete - request_id: " + reqID + " - " + err.Error())

		return response.WithError(ctx, err)
	}

	return response.WithMessage(ctx, fiber.StatusOK, res)
}

// GetByLocationID Field godoc
// @Summary Get fields by location ID
// @Description Get fields by location ID
// @Tags fields
// @Accept json
// @Produce json
// @Param location_id path string true "Location ID"
// @Param pagination query gdto.PaginationRequest false "Pagination request"
// @Success 200 {object} response.Data[dto.GetFieldsResponse]
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /locations/{location_id}/fields [get]
func (h *Handler) GetByLocationID(ctx *fiber.Ctx) error {
	locationID := ctx.Params("location_id")
	if err := h.validator.Var(locationID, "required,uuid"); err != nil {
		err = failure.BadRequestFromString(err.Error())

		h.logger.Error(identifier, "get all - validate error: %w", err)

		return response.WithError(ctx, err)
	}

	var req gdto.PaginationRequest
	if err := ctx.QueryParser(&req); err != nil {
		err = failure.BadRequestFromString(err.Error())

		h.logger.Error(identifier, "get all - query parsing error: %w", err)

		return response.WithError(ctx, err)
	}

	if err := h.validator.Struct(req); err != nil {
		err = failure.BadRequestFromString(err.Error())

		h.logger.Error(identifier, "get all - validate error: %w", err)

		return response.WithError(ctx, err)
	}

	data, err := h.service.GetByLocationID(ctx.UserContext(), locationID, req)
	if err != nil {
		reqID := "unknown"
		if id, ok := ctx.Locals("request_id").(string); ok {
			reqID = id
		}

		h.logger.Error("http - field - get by location id - request_id: " + reqID + " - " + err.Error())

		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusOK, data)
}
