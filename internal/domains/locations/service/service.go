package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/savioruz/goth/config"
	"github.com/savioruz/goth/internal/domains/locations/dto"
	"github.com/savioruz/goth/internal/domains/locations/repository"
	"github.com/savioruz/goth/pkg/failure"
	"github.com/savioruz/goth/pkg/gdto"
	"github.com/savioruz/goth/pkg/helper"
	"github.com/savioruz/goth/pkg/logger"
	"github.com/savioruz/goth/pkg/postgres"
	"github.com/savioruz/goth/pkg/redis"
	"reflect"
	"strconv"
)

type LocationService interface {
	Create(ctx context.Context, req dto.CreateLocationRequest) (res string, err error)
	Get(ctx context.Context, id string) (res dto.LocationResponse, err error)
	Count(ctx context.Context, filter string) (res int, err error)
	GetAll(ctx context.Context, req gdto.PaginationRequest) (res dto.PaginatedLocationResponse, err error)
	Update(ctx context.Context, id string, req dto.UpdateLocationRequest) (res string, err error)
	Delete(ctx context.Context, id string) (res string, err error)
}

type locationService struct {
	db     postgres.PgxIface
	repo   repository.Querier
	cache  redis.IRedisCache
	cfg    *config.Config
	logger logger.Interface
}

func New(db postgres.PgxIface, repo repository.Querier, cache redis.IRedisCache, cfg *config.Config, l logger.Interface) LocationService {
	return &locationService{
		db:     db,
		repo:   repo,
		cache:  cache,
		cfg:    cfg,
		logger: l,
	}
}

const (
	cacheGetLocationsKey = "locations"
	cacheGetLocationKey  = "location"

	identifier = "service - location - %s"
)

func (s *locationService) Create(ctx context.Context, req dto.CreateLocationRequest) (res string, err error) {
	newLocation, err := s.repo.CreateLocation(ctx, s.db, repository.CreateLocationParams{
		Name:        req.Name,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		Description: helper.PgString(req.Description),
	})
	if err != nil {
		s.logger.Error(identifier, "create - failed to create location: %w", err)

		return res, err
	}

	res = newLocation.ID.String()

	go func() {
		ctx := context.WithoutCancel(ctx)

		if err := s.cache.Delete(ctx, helper.BuildCacheKey(cacheGetLocationsKey, "count")); err != nil {
			s.logger.Error(identifier, "create - failed to delete cache for count: %w", err)
		}

		if err := s.cache.Clear(ctx, helper.BuildCacheKey(cacheGetLocationsKey, "*")); err != nil {
			s.logger.Error(identifier, "create - failed to clear cache: %w", err)
		}
	}()

	return res, nil
}

func (s *locationService) Get(ctx context.Context, id string) (res dto.LocationResponse, err error) {
	cacheKey := helper.BuildCacheKey(cacheGetLocationKey, id)

	var cacheRes dto.LocationResponse
	err = s.cache.Get(ctx, cacheKey, &cacheRes)

	if err == nil {
		s.logger.Info(identifier, "get - location %s - cache hit", id)

		return cacheRes, nil
	}

	location, err := s.repo.GetLocationById(ctx, s.db, helper.PgUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		s.logger.Info(identifier, "get - location %s - not found", id)

		err := failure.NotFound(fmt.Sprintf("location %s - not found", id))

		return res, err
	}

	if err != nil {
		s.logger.Error(identifier, "get - failed to get location by id: %w", err)

		return res, err
	}

	res = res.FromModel(location)

	go func() {
		err := s.cache.Save(context.WithoutCancel(ctx), cacheKey, res, s.cfg.Cache.Duration)
		if err != nil {
			s.logger.Error(identifier, "get - failed to set cache: %w", err)
		}
	}()

	return res, nil
}

func (s *locationService) Count(ctx context.Context, filter string) (res int, err error) {
	cacheKey := helper.BuildCacheKey(cacheGetLocationsKey, "count")

	var cacheRes int
	err = s.cache.Get(ctx, cacheKey, &cacheRes)

	if err == nil {
		s.logger.Info(identifier, "count - cache hit")

		return cacheRes, nil
	}

	totalItems, err := s.repo.CountLocationsWithFilter(ctx, s.db, filter)
	if err != nil {
		s.logger.Error(identifier, "count - failed to count locations: %w", err)

		return res, err
	}

	res = int(totalItems)

	go func() {
		err := s.cache.Save(context.WithoutCancel(ctx), cacheKey, res, s.cfg.Cache.Duration)
		if err != nil {
			s.logger.Error(identifier, "count - failed to set cache: %w", err)
		}
	}()

	return res, nil
}

func (s *locationService) GetAll(ctx context.Context, req gdto.PaginationRequest) (res dto.PaginatedLocationResponse, err error) {
	page, limit := helper.DefaultPagination(req.Page, req.Limit)

	keyArgs := map[string]string{}
	keyArgs["page"] = strconv.Itoa(page)
	keyArgs["limit"] = strconv.Itoa(limit)
	keyArgs["filter"] = req.Filter
	cacheKey := helper.BuildCacheKey(cacheGetLocationsKey, helper.GenerateUniqueKey(keyArgs))

	var cacheRes dto.PaginatedLocationResponse

	err = s.cache.Get(ctx, cacheKey, &cacheRes)
	if err == nil {
		s.logger.Info(identifier, "get all - cache hit")

		return cacheRes, nil
	}

	totalItems, err := s.Count(ctx, req.Filter)
	if err != nil {
		s.logger.Error(identifier, "get all - failed to count locations: %w", err)

		return res, err
	}

	offset := helper.CalculateOffset(page, limit)

	locations, err := s.repo.GetLocationsWithFilter(ctx, s.db, repository.GetLocationsWithFilterParams{
		Column1: req.Filter,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		s.logger.Error(identifier, "get all - failed to get locations: %w", err)

		return res, err
	}

	res.FromModel(locations, totalItems, limit)

	go func() {
		err := s.cache.Save(context.WithoutCancel(ctx), cacheKey, res, s.cfg.Cache.Duration)
		if err != nil {
			s.logger.Error(identifier, "get all - failed to set cache: %w", err)
		}
	}()

	return res, nil
}

func (s *locationService) Update(ctx context.Context, id string, req dto.UpdateLocationRequest) (res string, err error) {
	existingLocation, err := s.repo.GetLocationById(ctx, s.db, helper.PgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = failure.NotFound(fmt.Sprintf("location %s - not found", id))
		}

		s.logger.Error(identifier, "update - failed to get location by id: %w", err)

		return res, err
	}

	val := reflect.ValueOf(req)
	typ := reflect.TypeOf(req)

	var updateFields []string

	for i := range val.NumField() {
		field := val.Field(i)
		if field.IsZero() {
			continue
		}

		fieldName := typ.Field(i).Tag.Get("json")
		updateFields = append(updateFields, fieldName)

		switch fieldName {
		case "name":
			existingLocation.Name = field.Interface().(string)
		case "latitude":
			existingLocation.Latitude = field.Interface().(float64)
		case "longitude":
			existingLocation.Longitude = field.Interface().(float64)
		case "description":
			existingLocation.Description = helper.PgString(field.Interface().(string))
		}
	}

	if len(updateFields) == 0 {
		s.logger.Error(identifier, "update - at least one field is required to update")

		err := failure.BadRequestFromString("at least one field is required to update")

		return res, err
	}

	newLocation, err := s.repo.UpdateLocation(ctx, s.db, repository.UpdateLocationParams{
		ID:          helper.PgUUID(id),
		Name:        existingLocation.Name,
		Latitude:    existingLocation.Latitude,
		Longitude:   existingLocation.Longitude,
		Description: existingLocation.Description,
	})

	if err != nil {
		s.logger.Error(identifier, "update - failed to update location: %w", err)

		return res, err
	}

	res = newLocation.ID.String()

	go func() {
		ctx := context.WithoutCancel(ctx)

		if err := s.cache.Delete(ctx, helper.BuildCacheKey(cacheGetLocationsKey, "count")); err != nil {
			s.logger.Error(identifier, "update - failed to delete cache for count: %w", err)
		}

		if err := s.cache.Delete(ctx, helper.BuildCacheKey(cacheGetLocationKey, id)); err != nil {
			s.logger.Error(identifier, "update - failed to delete cache for location %s: %w", id, err)
		}

		if err := s.cache.Clear(ctx, helper.BuildCacheKey(cacheGetLocationsKey, "*")); err != nil {
			s.logger.Error(identifier, "update - failed to clear cache: %w", err)
		}
	}()

	return res, nil
}

func (s *locationService) Delete(ctx context.Context, id string) (res string, err error) {
	existingLocation, err := s.repo.DeleteLocation(ctx, s.db, helper.PgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = failure.NotFound(fmt.Sprintf("location %s - not found", id))
		}

		s.logger.Error(identifier, "delete - failed to get location by id: %w", err)

		return res, err
	}

	res = existingLocation.ID.String()

	go func() {
		ctx := context.WithoutCancel(ctx)

		if err := s.cache.Delete(ctx, helper.BuildCacheKey(cacheGetLocationsKey, "count")); err != nil {
			s.logger.Error(identifier, "delete - failed to delete cache for count: %w", err)
		}

		if err := s.cache.Delete(ctx, helper.BuildCacheKey(cacheGetLocationKey, id)); err != nil {
			s.logger.Error(identifier, "delete - failed to delete cache for location %s: %w", id, err)
		}

		if err := s.cache.Clear(ctx, helper.BuildCacheKey(cacheGetLocationsKey, "*")); err != nil {
			s.logger.Error(identifier, "delete - failed to clear cache: %w", err)
		}
	}()

	return res, nil
}
