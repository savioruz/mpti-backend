package service

import (
	"context"
	"fmt"
	"time"

	"github.com/savioruz/goth/config"
	"github.com/savioruz/goth/internal/domains/user/dto"
	"github.com/savioruz/goth/internal/domains/user/repository"
	"github.com/savioruz/goth/pkg/failure"
	"github.com/savioruz/goth/pkg/logger"
	"github.com/savioruz/goth/pkg/postgres"
	"github.com/savioruz/goth/pkg/redis"
)

type UserService interface {
	Profile(ctx context.Context, email string) (res dto.UserProfileResponse, err error)
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
