package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/savioruz/goth/internal/delivery/http/middleware"
	"github.com/savioruz/goth/internal/delivery/http/response"

	"github.com/savioruz/goth/internal/domains/user/dto"
	"github.com/savioruz/goth/internal/domains/user/service"
	"github.com/savioruz/goth/pkg/failure"
	"github.com/savioruz/goth/pkg/logger"
)

var (
	ErrEmailNil       = errors.New("email is nil")
	ErrEmailNotString = errors.New("email is not string")
)

type Handler struct {
	service   service.UserService
	logger    logger.Interface
	validator *validator.Validate
}

func New(s service.UserService, l logger.Interface, v *validator.Validate) *Handler {
	return &Handler{
		service:   s,
		logger:    l,
		validator: v,
	}
}

func (h *Handler) RegisterRoutes(r fiber.Router) {
	users := r.Group("/users")

	users.Get("/profile", middleware.Jwt(), h.Profile)

	// Admin routes - only accessible by admin
	users.Get("/admin", middleware.Jwt(), middleware.AdminOnly(), h.GetAllUsers)
	users.Get("/admin/:id", middleware.Jwt(), middleware.AdminOnly(), h.GetUserByID)
	users.Patch("/admin/:id/role", middleware.Jwt(), middleware.AdminOnly(), h.UpdateUserRole)
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

// GetAllUsers godoc
// @Summary Get all users (Admin only)
// @Description Get all users with pagination and filtering
// @Tags users
// @Accept json
// @Produce json
// @Param email query string false "Filter by email"
// @Param full_name query string false "Filter by full name"
// @Param level query string false "Filter by user level"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10, max: 100)"
// @Success 200 {object} response.Data[dto.PaginatedUserResponse]
// @Failure 400 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /users/admin [get]
// @Security BearerAuth
func (h *Handler) GetAllUsers(ctx *fiber.Ctx) error {
	var req dto.GetUsersRequest

	if err := ctx.QueryParser(&req); err != nil {
		h.logger.Error("http - user - GetAllUsers - query parser error: %v", err)
		return response.WithError(ctx, err)
	}

	users, err := h.service.GetAllUsers(ctx.Context(), req)
	if err != nil {
		h.logger.Error("http - user - GetAllUsers - service error: %v", err)
		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusOK, users)
}

// GetUserByID godoc
// @Summary Get user by ID (Admin only)
// @Description Get user details by ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} response.Data[dto.UserAdminResponse]
// @Failure 400 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /users/admin/{id} [get]
// @Security BearerAuth
func (h *Handler) GetUserByID(ctx *fiber.Ctx) error {
	userID := ctx.Params("id")

	if userID == "" {
		h.logger.Error("http - user - GetUserByID - user ID is required")
		return response.WithError(ctx, failure.BadRequestFromString("user ID is required"))
	}

	user, err := h.service.GetUserByID(ctx.Context(), userID)
	if err != nil {
		h.logger.Error("http - user - GetUserByID - service error: %v", err)
		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusOK, user)
}

// UpdateUserRole godoc
// @Summary Update user role (Admin only)
// @Description Update user role/level
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param role body dto.UpdateUserRoleRequest true "Update role request"
// @Success 200 {object} response.Data[dto.UserAdminResponse]
// @Failure 400 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /users/admin/{id}/role [patch]
// @Security BearerAuth
func (h *Handler) UpdateUserRole(ctx *fiber.Ctx) error {
	userID := ctx.Params("id")

	if userID == "" {
		h.logger.Error("http - user - UpdateUserRole - user ID is required")
		return response.WithError(ctx, failure.BadRequestFromString("user ID is required"))
	}

	var req dto.UpdateUserRoleRequest
	if err := ctx.BodyParser(&req); err != nil {
		h.logger.Error("http - user - UpdateUserRole - body parser error: %v", err)
		return response.WithError(ctx, err)
	}

	if err := h.validator.Struct(req); err != nil {
		validationErr := err.Error()
		transformErr := failure.BadRequestFromString(validationErr)
		h.logger.Error("http - user - UpdateUserRole - validation error: %v", transformErr)
		return response.WithError(ctx, transformErr)
	}

	user, err := h.service.UpdateUserRole(ctx.Context(), userID, req)
	if err != nil {
		h.logger.Error("http - user - UpdateUserRole - service error: %v", err)
		return response.WithError(ctx, err)
	}

	return response.WithJSON(ctx, fiber.StatusOK, user)
}
