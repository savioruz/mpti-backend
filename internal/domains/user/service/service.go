package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/savioruz/goth/config"
	"github.com/savioruz/goth/internal/domains/user/dto"
	"github.com/savioruz/goth/internal/domains/user/repository"
	"github.com/savioruz/goth/pkg/failure"
	"github.com/savioruz/goth/pkg/helper"
	"github.com/savioruz/goth/pkg/logger"
	"github.com/savioruz/goth/pkg/postgres"
	"github.com/savioruz/goth/pkg/redis"
)

type UserService interface {
	Profile(ctx context.Context, email string) (res dto.UserProfileResponse, err error)
	GetAllUsers(ctx context.Context, req dto.GetUsersRequest) (dto.PaginatedUserResponse, error)
	GetUserByID(ctx context.Context, userID string) (dto.UserAdminResponse, error)
	UpdateUserRole(ctx context.Context, id string, req dto.UpdateUserRoleRequest) (dto.UserAdminResponse, error)
}

const (
	cacheGetUserKey     = "cache:get_user:%s"
	defaultCacheTimeout = 5 * time.Second
)

type userService struct {
	db     postgres.PgxIface
	repo   repository.Querier
	cache  redis.IRedisCache
	config *config.Config
	logger logger.Interface
}

func New(db postgres.PgxIface, repo repository.Querier, cache redis.IRedisCache, cfg *config.Config, l logger.Interface) UserService {
	return &userService{
		db:     db,
		repo:   repo,
		cache:  cache,
		config: cfg,
		logger: l,
	}
}

func (s *userService) Profile(ctx context.Context, email string) (res dto.UserProfileResponse, err error) {
	cacheKey := fmt.Sprintf(cacheGetUserKey, email)

	var cacheRes dto.UserProfileResponse
	err = s.cache.Get(ctx, cacheKey, &cacheRes)

	if err == nil {
		s.logger.Info("service - user %s - profile - cache hit", email)

		return cacheRes, nil
	}

	user, err := s.repo.GetUserByEmail(ctx, s.db, email)
	if err != nil {
		s.logger.Error("service - user - profile - failed to get user by email", err)

		return dto.UserProfileResponse{}, failure.InternalError(err)
	}

	if user == (repository.User{}) {
		s.logger.Error("service - user - profile - user not found")

		return dto.UserProfileResponse{}, failure.NotFound("user not found")
	}

	var profileResponse dto.UserProfileResponse
	profileResponse = profileResponse.ToProfileResponse(user)

	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), defaultCacheTimeout)
		defer cancel()

		cacheErr := s.cache.Save(cacheCtx, cacheKey, profileResponse, s.config.Cache.Duration)
		if cacheErr != nil {
			s.logger.Error("service - user - profile - failed to set cache", cacheErr)
		}
	}()

	return profileResponse, nil
}

func (s *userService) GetAllUsers(ctx context.Context, req dto.GetUsersRequest) (res dto.PaginatedUserResponse, err error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	offset := (page - 1) * limit

	totalCount, err := s.repo.CountUsers(ctx, s.db, repository.CountUsersParams{
		Column1: req.Email,
		Column2: req.FullName,
		Column3: req.Level,
	})
	if err != nil {
		s.logger.Error("service - user - GetAllUsers - failed to count users: %v", err)

		return res, failure.InternalError(err)
	}

	// Get paginated users
	users, err := s.repo.GetAllUsers(ctx, s.db, repository.GetAllUsersParams{
		Column1: req.Email,
		Column2: req.FullName,
		Column3: req.Level,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		s.logger.Error("service - user - GetAllUsers - failed to get users: %v", err)

		return res, failure.InternalError(err)
	}

	res.FromModel(users, int(totalCount), limit)

	return res, nil
}

func (s *userService) GetUserByID(ctx context.Context, userID string) (res dto.UserAdminResponse, err error) {
	user, err := s.repo.GetUserByID(ctx, s.db, helper.PgUUID(userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logger.Error("service - user - GetUserByID - user not found: %s", userID)

			return res, failure.NotFound("user not found")
		}

		s.logger.Error("service - user - GetUserByID - failed to get user: %v", err)

		return res, failure.InternalError(err)
	}

	res = dto.UserAdminResponse{}.FromModel(user)

	return res, nil
}

func (s *userService) UpdateUserRole(ctx context.Context, id string, req dto.UpdateUserRoleRequest) (res dto.UserAdminResponse, err error) {
	user, err := s.repo.UpdateUserRole(ctx, s.db, repository.UpdateUserRoleParams{
		ID:    helper.PgUUID(id),
		Level: req.Level,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logger.Error("service - user - UpdateUserRole - user not found: %s", id)

			return res, failure.NotFound("user not found")
		}

		s.logger.Error("service - user - UpdateUserRole - failed to update user role: %v", err)

		return res, failure.InternalError(err)
	}

	res = dto.UserAdminResponse{}.FromModel(user)

	return res, nil
}
