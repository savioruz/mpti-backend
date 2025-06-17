package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/savioruz/goth/config"
	"github.com/savioruz/goth/internal/domains/fields/dto"
	"github.com/savioruz/goth/internal/domains/fields/repository"
	"github.com/savioruz/goth/pkg/failure"
	"github.com/savioruz/goth/pkg/gdto"
	"github.com/savioruz/goth/pkg/helper"
	"github.com/savioruz/goth/pkg/logger"
	"github.com/savioruz/goth/pkg/postgres"
	"github.com/savioruz/goth/pkg/redis"
	"reflect"
	"strconv"
)

type FieldService interface {
	Create(ctx context.Context, req dto.FieldCreateRequest) (string, error)
	Get(ctx context.Context, id string) (dto.FieldResponse, error)
	GetAll(ctx context.Context, req gdto.PaginationRequest) (dto.GetFieldsResponse, error)
	Count(ctx context.Context, req gdto.PaginationRequest) (int, error)
	GetByLocationID(ctx context.Context, locationID string, req gdto.PaginationRequest) (dto.GetFieldsResponse, error)
	CountByLocationID(ctx context.Context, locationID string, req gdto.PaginationRequest) (int, error)
	Update(ctx context.Context, id string, req dto.FieldUpdateRequest) (string, error)
	Delete(ctx context.Context, id string) (string, error)
}

type fieldService struct {
	db     postgres.PgxIface
	repo   repository.Querier
	cache  redis.IRedisCache
	cfg    *config.Config
	logger logger.Interface
}

func New(db postgres.PgxIface, repo repository.Querier, cache redis.IRedisCache, cfg *config.Config, l logger.Interface) FieldService {
	return &fieldService{
		db:     db,
		repo:   repo,
		cache:  cache,
		cfg:    cfg,
		logger: l,
	}
}

const (
	cacheGetFieldsKey   = "fields"
	cacheCountFieldsKey = "fields:count"
	cacheGetFieldKey    = "field"

	identifier = "service - location - %s"
)

func (s *fieldService) Create(ctx context.Context, req dto.FieldCreateRequest) (res string, err error) {
	newField, err := s.repo.CreateField(ctx, s.db, repository.CreateFieldParams{
		LocationID:  helper.PgUUID(req.LocationID.String()),
		Name:        req.Name,
		Type:        req.Type,
		Price:       helper.PgInt64(req.Price),
		Description: helper.PgString(req.Description),
	})
	if err != nil {
		s.logger.Error(identifier, "create - failed to create field: %w", err)

		return res, err
	}

	res = newField.String()

	go func() {
		ctx := context.WithoutCancel(ctx)

		if err := s.cache.Clear(ctx, helper.BuildCacheKey(cacheCountFieldsKey, "*")); err != nil {
			s.logger.Error(identifier, "create - failed to delete cache: %w", err)
		}

		if err := s.cache.Clear(ctx, helper.BuildCacheKey(cacheGetFieldsKey, "*")); err != nil {
			s.logger.Error(identifier, "create - failed to clear cache: %w", err)
		}
	}()

	return res, nil
}

func (s *fieldService) Get(ctx context.Context, id string) (res dto.FieldResponse, err error) {
	cacheKey := helper.BuildCacheKey(cacheGetFieldKey, id)

	if err = s.cache.Get(ctx, cacheKey, &res); err == nil {
		return res, nil
	}

	field, err := s.repo.GetFieldById(ctx, s.db, helper.PgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = failure.NotFound(fmt.Sprintf("field %s - not found", id))
			s.logger.Error(identifier, "get - field not found: %w", err)

			return res, err
		}

		s.logger.Error(identifier, "get - failed get field: %w", err)

		return res, err
	}

	res = res.FromModel(field)

	go func() {
		err := s.cache.Save(context.WithoutCancel(ctx), cacheKey, res, s.cfg.Cache.Duration)
		if err != nil {
			s.logger.Error(identifier, "get - failed save cache: %w", err)
		}
	}()

	return res, nil
}

func (s *fieldService) GetAll(ctx context.Context, req gdto.PaginationRequest) (res dto.GetFieldsResponse, err error) {
	page, limit := helper.DefaultPagination(req.Page, req.Limit)

	keyArgs := map[string]string{}
	keyArgs["page"] = strconv.Itoa(page)
	keyArgs["limit"] = strconv.Itoa(limit)
	keyArgs["filter"] = req.Filter
	cacheKey := helper.BuildCacheKey(cacheGetFieldsKey, helper.GenerateUniqueKey(keyArgs))

	var cacheRes dto.GetFieldsResponse

	err = s.cache.Get(ctx, cacheKey, &cacheRes)
	if err == nil {
		s.logger.Info(identifier, "getAll - cache hit for filter %s", req.Filter)

		return cacheRes, nil
	}

	totalItems, err := s.Count(ctx, req)
	if err != nil {
		s.logger.Error(identifier, "getAll - failed to count fields: %w", err)

		return res, err
	}

	offset := helper.CalculateOffset(page, limit)

	fields, err := s.repo.GetFields(ctx, s.db, repository.GetFieldsParams{
		Column1: req.Filter,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		s.logger.Error(identifier, "getAll - failed to get fields: %w", err)

		return res, nil
	}

	res.FromModel(fields, totalItems, limit)

	go func() {
		ctx := context.WithoutCancel(ctx)

		err := s.cache.Save(ctx, cacheKey, res, s.cfg.Cache.Duration)
		if err != nil {
			s.logger.Error(identifier, "getAll - failed to set cache: %w", err)
		}
	}()

	return res, nil
}

func (s *fieldService) Count(ctx context.Context, req gdto.PaginationRequest) (total int, err error) {
	page, limit := helper.DefaultPagination(req.Page, req.Limit)

	keyArgs := map[string]string{}
	keyArgs["page"] = strconv.Itoa(page)
	keyArgs["limit"] = strconv.Itoa(limit)
	keyArgs["filter"] = req.Filter
	cacheKey := helper.BuildCacheKey(cacheCountFieldsKey, helper.GenerateUniqueKey(keyArgs))

	var cacheRes int

	err = s.cache.Get(ctx, cacheKey, &cacheRes)
	if err == nil {
		s.logger.Info(identifier, "count - cache hit for filter %s", req.Filter)

		return cacheRes, nil
	}

	totalItems, err := s.repo.CountFields(ctx, s.db, req.Filter)
	if err != nil {
		s.logger.Error(identifier, "count - failed to count fields: %w", err)

		return total, err
	}

	total = int(totalItems)

	go func() {
		ctx := context.WithoutCancel(ctx)

		err := s.cache.Save(ctx, cacheKey, total, s.cfg.Cache.Duration)
		if err != nil {
			s.logger.Error(identifier, "count - failed to set cache: %w", err)
		}
	}()

	return total, nil
}

func (s *fieldService) GetByLocationID(ctx context.Context, locationID string, req gdto.PaginationRequest) (res dto.GetFieldsResponse, err error) {
	page, limit := helper.DefaultPagination(req.Page, req.Limit)

	keyArgs := map[string]string{}
	keyArgs["page"] = strconv.Itoa(page)
	keyArgs["limit"] = strconv.Itoa(limit)
	keyArgs["filter"] = req.Filter
	cacheKey := helper.BuildCacheKey(cacheGetFieldsKey, helper.GenerateUniqueKey(keyArgs))

	var cacheRes dto.GetFieldsResponse

	err = s.cache.Get(ctx, cacheKey, &cacheRes)
	if err == nil {
		s.logger.Info(identifier, "getByLocationID - cache hit for location %s with filter %s", locationID, req.Filter)

		return cacheRes, nil
	}

	totalItems, err := s.CountByLocationID(ctx, locationID, req)
	if err != nil {
		s.logger.Error(identifier, "getByLocationID - failed to count fields: %w", err)

		return res, err
	}

	offset := helper.CalculateOffset(page, limit)

	fields, err := s.repo.GetFieldsByLocationID(ctx, s.db, repository.GetFieldsByLocationIDParams{
		Column1:    req.Filter,
		LocationID: helper.PgUUID(locationID),
		Limit:      int32(limit),
		Offset:     int32(offset),
	})
	if err != nil {
		s.logger.Error(identifier, "getByLocationID - failed to get fields: %w", err)

		return res, nil
	}

	res.FromModel(fields, totalItems, limit)

	go func() {
		if err := s.cache.Save(context.WithoutCancel(ctx), cacheKey, res, s.cfg.Cache.Duration); err != nil {
			s.logger.Error(identifier, "get by location id - failed to save cache: %w", err)
		}
	}()

	return res, nil
}

func (s *fieldService) CountByLocationID(ctx context.Context, locationID string, req gdto.PaginationRequest) (res int, err error) {
	page, limit := helper.DefaultPagination(req.Page, req.Limit)

	keyArgs := map[string]string{}
	keyArgs["location_id"] = locationID
	keyArgs["page"] = strconv.Itoa(page)
	keyArgs["limit"] = strconv.Itoa(limit)
	keyArgs["filter"] = req.Filter
	cacheKey := helper.BuildCacheKey(cacheCountFieldsKey, helper.GenerateUniqueKey(keyArgs))

	var cacheRes int

	err = s.cache.Get(ctx, cacheKey, &cacheRes)
	if err == nil {
		s.logger.Info(identifier, "countByLocationID - cache hit for location %s with filter %s", locationID, req.Filter)

		return cacheRes, nil
	}

	totalItems, err := s.repo.CountFieldsByLocationID(ctx, s.db, repository.CountFieldsByLocationIDParams{
		Column1:    req.Filter,
		LocationID: helper.PgUUID(locationID),
	})
	if err != nil {
		s.logger.Error(identifier, "countByLocationID - failed to count fields: %w", err)

		return res, err
	}

	res = int(totalItems)

	go func() {
		ctx := context.WithoutCancel(ctx)

		err := s.cache.Save(ctx, cacheKey, res, s.cfg.Cache.Duration)
		if err != nil {
			s.logger.Error(identifier, "countByLocationID - failed to set cache: %w", err)
		}
	}()

	return res, nil
}

func (s *fieldService) Update(ctx context.Context, id string, req dto.FieldUpdateRequest) (res string, err error) {
	existingField, err := s.repo.GetFieldById(ctx, s.db, helper.PgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = failure.NotFound(fmt.Sprintf("fields %s - not found", id))
		}

		s.logger.Error(identifier, "update - failed get field: %w", err)

		return res, err
	}

	val := reflect.ValueOf(req)
	typ := reflect.TypeOf(req)

	var updatedFields []string

	for i := range val.NumField() {
		field := val.Field(i)
		if field.IsZero() {
			continue
		}

		fieldName := typ.Field(i).Tag.Get("json")
		updatedFields = append(updatedFields, fieldName)

		switch fieldName {
		case "location_id":
			existingField.LocationID = helper.PgUUID(field.Interface().(string))
		case "name":
			existingField.Name = field.Interface().(string)
		case "type":
			existingField.Type = field.Interface().(string)
		case "price":
			existingField.Price = helper.PgInt64(field.Int())
		case "description":
			existingField.Description = helper.PgString(field.Interface().(string))
		}
	}

	if len(updatedFields) == 0 {
		s.logger.Error(identifier, "update - at least one field is required to update")

		err := failure.BadRequestFromString("at least one field is required to update")

		return res, err
	}

	newField, err := s.repo.UpdateField(ctx, s.db, repository.UpdateFieldParams{
		ID:          helper.PgUUID(id),
		LocationID:  existingField.LocationID,
		Name:        existingField.Name,
		Type:        existingField.Type,
		Price:       existingField.Price,
		Description: existingField.Description,
	})

	if err != nil {
		s.logger.Error(identifier, "update - failed to update field: %w", err)

		return res, err
	}

	res = newField.String()

	go func() {
		ctx := context.WithoutCancel(ctx)

		if err := s.cache.Delete(ctx, helper.BuildCacheKey(cacheGetFieldKey, id)); err != nil {
			s.logger.Error(identifier, "update - failed to delete cache: %w", err)
		}

		if err := s.cache.Clear(ctx, helper.BuildCacheKey(cacheCountFieldsKey, "*")); err != nil {
			s.logger.Error(identifier, "update - failed to delete cache: %w", err)
		}

		if err := s.cache.Clear(ctx, helper.BuildCacheKey(cacheGetFieldsKey, "*")); err != nil {
			s.logger.Error(identifier, "update - failed to clear cache: %w", err)
		}
	}()

	return res, nil
}

func (s *fieldService) Delete(ctx context.Context, id string) (res string, err error) {
	existingField, err := s.repo.DeleteField(ctx, s.db, helper.PgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = failure.NotFound(fmt.Sprintf("fields %s - not found", id))
		}

		s.logger.Error(identifier, "delete - failed get field: %w", err)

		return res, err
	}

	res = existingField.String()

	go func() {
		ctx := context.WithoutCancel(ctx)

		if err := s.cache.Delete(ctx, helper.BuildCacheKey(cacheGetFieldKey, id)); err != nil {
			s.logger.Error(identifier, "delete - failed to delete cache: %w", err)
		}

		if err := s.cache.Clear(ctx, helper.BuildCacheKey(cacheGetFieldsKey, "*")); err != nil {
			s.logger.Error(identifier, "delete - failed to delete cache: %w", err)
		}

		if err := s.cache.Clear(ctx, helper.BuildCacheKey(cacheGetFieldsKey, "*")); err != nil {
			s.logger.Error(identifier, "delete - failed to clear cache: %w", err)
		}
	}()

	return res, nil
}
