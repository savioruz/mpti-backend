package service

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"reflect"
	"strconv"

	"github.com/jackc/pgx/v5/pgconn"
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
	"github.com/savioruz/goth/pkg/supabase"
)

type FieldService interface {
	Create(ctx context.Context, req dto.FieldCreateRequest) (string, error)
	Get(ctx context.Context, id string) (dto.FieldResponse, error)
	GetAll(ctx context.Context, req gdto.PaginationRequest) (dto.GetFieldsResponse, error)
	Count(ctx context.Context, req gdto.PaginationRequest) (int, error)
	GetByLocationID(ctx context.Context, locationID string, req gdto.PaginationRequest) (dto.GetFieldsResponse, error)
	CountByLocationID(ctx context.Context, locationID string, req gdto.PaginationRequest) (int, error)
	Update(ctx context.Context, id string, req dto.FieldUpdateRequest) (string, error)
	Delete(ctx context.Context, id string) error
	UploadImages(ctx context.Context, fieldID string, files []*multipart.FileHeader) ([]string, error)
	DeleteImage(ctx context.Context, fieldID, imageURL string) error
}

type fieldService struct {
	db            postgres.PgxIface
	repo          repository.Querier
	cache         redis.IRedisCache
	cfg           *config.Config
	logger        logger.Interface
	storageClient *supabase.Client
}

func New(db postgres.PgxIface, repo repository.Querier, cache redis.IRedisCache, cfg *config.Config, l logger.Interface, storageClient *supabase.Client) FieldService {
	return &fieldService{
		db:            db,
		repo:          repo,
		cache:         cache,
		cfg:           cfg,
		logger:        l,
		storageClient: storageClient,
	}
}

const (
	cacheGetFieldsKey   = "fields"
	cacheCountFieldsKey = "fields:count"
	cacheGetFieldKey    = "field"

	identifier = "service - field - %s"

	// Upload constants
	MaxFileSize       = 10 << 20 // 10MB
	MaxFilesPerUpload = 10
)

func (s *fieldService) Create(ctx context.Context, req dto.FieldCreateRequest) (res string, err error) {
	newField, err := s.repo.CreateField(ctx, s.db, repository.CreateFieldParams{
		LocationID:  helper.PgUUID(req.LocationID.String()),
		Name:        req.Name,
		Type:        req.Type,
		Price:       helper.PgInt64(req.Price),
		Description: helper.PgString(req.Description),
		Images:      req.Images,
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
		case "images":
			existingField.Images = field.Interface().([]string)
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
		Images:      existingField.Images,
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

func (s *fieldService) Delete(ctx context.Context, id string) (err error) {
	existingField, err := s.repo.GetFieldById(ctx, s.db, helper.PgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = failure.NotFound(fmt.Sprintf("fields %s - not found", id))
		}

		s.logger.Error(identifier, "delete - failed to get field: %w", err)

		return err
	}

	err = s.repo.DeleteField(ctx, s.db, helper.PgUUID(id))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			err = failure.Conflict("field used by other entities")
		}

		s.logger.Error(identifier, "delete - failed to delete field: %w", err)

		return err
	}

	go func() {
		ctx := context.WithoutCancel(ctx)

		// Delete images from storage
		for _, imageURL := range existingField.Images {
			if deleteErr := s.storageClient.DeleteFile(ctx, imageURL); deleteErr != nil {
				s.logger.Error(identifier, "delete - failed to delete image %s from storage: %w", imageURL, deleteErr)
			}
		}

		// Clear cache
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

	return nil
}

func (s *fieldService) UploadImages(ctx context.Context, fieldID string, files []*multipart.FileHeader) (urls []string, err error) {
	if len(files) == 0 {
		err = failure.BadRequestFromString("no files uploaded")
		s.logger.Error(identifier, "uploadImages - no files uploaded: %w", err)

		return urls, err
	}

	if len(files) > MaxFilesPerUpload {
		err = failure.BadRequestFromString(fmt.Sprintf("maximum %d files allowed per upload", MaxFilesPerUpload))

		s.logger.Error(identifier, "uploadImages - too many files: %d", len(files))

		return urls, err
	}

	// Get existing field to append new images
	existingField, err := s.repo.GetFieldById(ctx, s.db, helper.PgUUID(fieldID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = failure.NotFound(fmt.Sprintf("field %s - not found", fieldID))
		}

		s.logger.Error(identifier, "uploadImages - failed to get field: %w", err)

		return urls, err
	}

	var uploadedURLs []string

	for _, file := range files {
		// Check file size
		if file.Size > MaxFileSize {
			err = failure.BadRequestFromString(fmt.Sprintf("file %s exceeds maximum size of %d bytes", file.Filename, MaxFileSize))
			s.logger.Error(identifier, "uploadImages - file too large: %s (%d bytes)", file.Filename, file.Size)

			return urls, err
		}

		fileHandle, err := file.Open()
		if err != nil {
			s.logger.Error(identifier, "uploadImages - failed to open file %s: %w", file.Filename, err)

			return urls, err
		}

		url, err := s.storageClient.UploadFile(ctx, fileHandle, file.Filename)
		fileHandle.Close() // Close the file handle immediately after use

		if err != nil {
			s.logger.Error(identifier, "uploadImages - failed to upload file %s: %w", file.Filename, err)

			return urls, err
		}

		uploadedURLs = append(uploadedURLs, url)
	}

	// Update field with new images (append to existing images)
	existingField.Images = append(existingField.Images, uploadedURLs...)

	if existingField.Images == nil {
		existingField.Images = []string{}
	}

	_, err = s.repo.UpdateField(ctx, s.db, repository.UpdateFieldParams{
		ID:          existingField.ID,
		LocationID:  existingField.LocationID,
		Name:        existingField.Name,
		Type:        existingField.Type,
		Price:       existingField.Price,
		Description: existingField.Description,
		Images:      existingField.Images,
	})
	if err != nil {
		s.logger.Error(identifier, "uploadImages - failed to update field with new images: %w", err)

		// Clean up uploaded files if database update fails
		for _, url := range uploadedURLs {
			if deleteErr := s.storageClient.DeleteFile(ctx, url); deleteErr != nil {
				s.logger.Error(identifier, "uploadImages - failed to cleanup uploaded file %s: %w", url, deleteErr)
			}
		}

		return urls, err
	}

	go func() {
		ctx := context.WithoutCancel(ctx)

		if err := s.cache.Clear(ctx, helper.BuildCacheKey(cacheGetFieldKey, "*")); err != nil {
			s.logger.Error(identifier, "uploadImages - failed to clear cache: %w", err)
		}

		if err := s.cache.Clear(ctx, helper.BuildCacheKey(cacheGetFieldsKey, "*")); err != nil {
			s.logger.Error(identifier, "uploadImages - failed to clear cache: %w", err)
		}
	}()

	return uploadedURLs, nil
}

func (s *fieldService) DeleteImage(ctx context.Context, fieldID, imageURL string) error {
	// Get existing field to remove image from the array
	existingField, err := s.repo.GetFieldById(ctx, s.db, helper.PgUUID(fieldID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = failure.NotFound(fmt.Sprintf("field %s - not found", fieldID))
		}

		s.logger.Error(identifier, "deleteImage - failed to get field: %w", err)

		return err
	}

	// Find and remove the image URL from the array
	updatedImages := make([]string, 0) // Initialize as empty slice instead of nil
	found := false

	for _, img := range existingField.Images {
		if img != imageURL {
			updatedImages = append(updatedImages, img)
		} else {
			found = true
		}
	}

	if !found {
		err = failure.NotFound(fmt.Sprintf("image URL %s not found in field %s", imageURL, fieldID))
		s.logger.Error(identifier, "deleteImage - image URL not found: %w", err)

		return err
	}

	// Delete file from storage
	err = s.storageClient.DeleteFile(ctx, imageURL)
	if err != nil {
		s.logger.Error(identifier, "deleteImage - failed to delete file %s: %w", imageURL, err)

		return err
	}

	// Update field with removed image
	_, err = s.repo.UpdateField(ctx, s.db, repository.UpdateFieldParams{
		ID:          existingField.ID,
		LocationID:  existingField.LocationID,
		Name:        existingField.Name,
		Type:        existingField.Type,
		Price:       existingField.Price,
		Description: existingField.Description,
		Images:      updatedImages,
	})
	if err != nil {
		s.logger.Error(identifier, "deleteImage - failed to update field after removing image: %w", err)

		return err
	}

	go func() {
		ctx := context.WithoutCancel(ctx)

		if err := s.cache.Clear(ctx, helper.BuildCacheKey(cacheGetFieldKey, "*")); err != nil {
			s.logger.Error(identifier, "deleteImage - failed to clear cache: %w", err)
		}

		if err := s.cache.Clear(ctx, helper.BuildCacheKey(cacheGetFieldsKey, "*")); err != nil {
			s.logger.Error(identifier, "deleteImage - failed to clear cache: %w", err)
		}
	}()

	return nil
}
