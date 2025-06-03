package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/savioruz/goth/internal/delivery/http/middleware"
	"github.com/savioruz/goth/internal/delivery/http/response"
	"github.com/savioruz/goth/internal/domains/locations/dto"
	"github.com/savioruz/goth/internal/domains/locations/service"
	"github.com/savioruz/goth/pkg/constant"
	"github.com/savioruz/goth/pkg/failure"
	"github.com/savioruz/goth/pkg/gdto"
	"github.com/savioruz/goth/pkg/logger"
)

type Handler struct {
	service   service.LocationService
	logger    logger.Interface
	validator *validator.Validate
}

func New(s service.LocationService, l logger.Interface, v *validator.Validate) *Handler {
	return &Handler{
		service:   s,
		logger:    l,
		validator: v,
	}
}

const (
	identifier = "http - location - %s"

	routePath = "/locations"
)

func (h *Handler) RegisterRoutes(r fiber.Router) {
	locations := r.Group(routePath)

	locations.Post("/", middleware.Jwt(), middleware.AdminOnly(), h.Create)
	locations.Get("/:id", h.Get)
	locations.Get("/", h.GetAll)
	locations.Patch("/:id", middleware.Jwt(), middleware.AdminOnly(), h.Update)
	locations.Delete("/:id", middleware.Jwt(), middleware.AdminOnly(), h.Delete)
}

// Create Location godoc
// @Summary Create new location
// @Description Create new location
// @Tags locations
// @Accept json
// @Produce json
// @Param location body dto.CreateLocationRequest true "Location create request"
// @Success 201 {object} response.Data[string]
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /locations/ [post]
// @Security BearerAuth
func (h *Handler) Create(ctx *fiber.Ctx) error {
	var req dto.CreateLocationRequest
	if err := ctx.BodyParser(&req); err != nil {
		err = failure.BadRequestFromString(err.Error())

		h.logger.Error(identifier, "create - body parsing error: %w", err)

		return response.WithError(ctx, err)
	}

	if err := h.validator.Struct(req); err != nil {
		err = failure.BadRequestFromString(err.Error())

		h.logger.Error(identifier, "create - validate error: %w", err)

		return response.WithError(ctx, err)
	}

	res, err := h.service.Create(ctx.UserContext(), req)
	if err != nil {
		h.logger.Error(identifier, "create - failed to create location: %w", err)

		return response.WithError(ctx, err)
	}

	return response.WithMessage(ctx, fiber.StatusCreated, res)
}

// Get Location godoc
// @Summary Get location by id
// @Description Get location by id
// @Tags locations
// @Accept json
// @Produce json
// @Param id path string true "Location ID"
// @Success 200 {object} response.Data[dto.LocationResponse]
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /locations/{id} [get]
func (h *Handler) Get(ctx *fiber.Ctx) error {
	id := ctx.Params(constant.RequestParamID)

	if err := h.validator.Var(id, constant.RequestValidateUUID); err != nil {
		err = failure.BadRequestFromString("invalid location id format")

		h.logger.Error(identifier, "get - invalid location id format: %w", err)

		return response.WithError(ctx, err)
	}

	res, err := h.service.Get(ctx.UserContext(), id)
	if err != nil {
		h.logger.Error(identifier, "get - failed to get location: %w", err)

		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusOK, res)
}

// GetAll Locations godoc
// @Summary Get all locations
// @Description Get all locations
// @Tags locations
// @Accept json
// @Produce json
// @Param request query gdto.PaginationRequest false "Pagination request"
// @Success 200 {object} response.Data[dto.PaginatedLocationResponse[dto.LocationResponse]]
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /locations/ [get]
func (h *Handler) GetAll(ctx *fiber.Ctx) error {
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

	res, err := h.service.GetAll(ctx.UserContext(), req)
	if err != nil {
		h.logger.Error(identifier, "get all - failed to get locations: %w", err)

		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusOK, res)
}

// Update Location godoc
// @Summary Update location by id
// @Description Update location by id
// @Tags locations
// @Accept json
// @Produce json
// @Param id path string true "Location ID"
// @Param location body dto.UpdateLocationRequest true "Location update request"
// @Success 200 {object} response.Data[string]
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /locations/{id} [patch]
// @Security BearerAuth
func (h *Handler) Update(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		err := failure.BadRequestFromString("location id is required")

		h.logger.Error(identifier, "update - location id is required: %w", err)

		return response.WithError(ctx, err)
	}

	if err := h.validator.Var(id, "required,uuid"); err != nil {
		err = failure.BadRequestFromString("invalid location id format")

		h.logger.Error(identifier, "update - invalid location id format: %w", err)

		return response.WithError(ctx, err)
	}

	var req dto.UpdateLocationRequest
	if err := ctx.BodyParser(&req); err != nil {
		err = failure.BadRequestFromString(err.Error())

		h.logger.Error(identifier, "update - body parsing error: %w", err)

		return response.WithError(ctx, err)
	}

	if err := h.validator.Struct(req); err != nil {
		err = failure.BadRequestFromString(err.Error())

		h.logger.Error(identifier, "update - validate error: %w", err)

		return response.WithError(ctx, err)
	}

	res, err := h.service.Update(ctx.UserContext(), id, req)
	if err != nil {
		h.logger.Error(identifier, "update - failed to update location: %w", err)

		return response.WithError(ctx, err)
	}

	return response.WithMessage(ctx, fiber.StatusOK, res)
}

// Delete Location godoc
// @Summary Delete location by id
// @Description Delete location by id
// @Tags locations
// @Accept json
// @Produce json
// @Param id path string true "Location ID"
// @Success 200 {object} response.Data[string]
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /locations/{id} [delete]
// @Security BearerAuth
func (h *Handler) Delete(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		err := failure.BadRequestFromString("location id is required")

		h.logger.Error(identifier, "delete - location id is required: %w", err)

		return response.WithError(ctx, err)
	}

	if err := h.validator.Var(id, "required,uuid"); err != nil {
		err = failure.BadRequestFromString("invalid location id format")

		h.logger.Error(identifier, "delete - invalid location id format: %w", err)

		return response.WithError(ctx, err)
	}

	res, err := h.service.Delete(ctx.UserContext(), id)
	if err != nil {
		h.logger.Error(identifier, "delete - failed to delete location: %w", err)

		return response.WithError(ctx, err)
	}

	return response.WithMessage(ctx, fiber.StatusOK, res)
}
