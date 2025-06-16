package service

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/savioruz/goth/config"
	"github.com/savioruz/goth/internal/domains/bookings/dto"
	"github.com/savioruz/goth/internal/domains/bookings/repository"
	fieldRepo "github.com/savioruz/goth/internal/domains/fields/repository"
	paymentDto "github.com/savioruz/goth/internal/domains/payments/dto"
	"github.com/savioruz/goth/internal/domains/payments/service"
	"github.com/savioruz/goth/pkg/constant"
	"github.com/savioruz/goth/pkg/failure"
	"github.com/savioruz/goth/pkg/gdto"
	"github.com/savioruz/goth/pkg/helper"
	"github.com/savioruz/goth/pkg/logger"
	"github.com/savioruz/goth/pkg/postgres"
	"github.com/savioruz/goth/pkg/redis"
	"strconv"
)

type BookingService interface {
	CreateBooking(ctx context.Context, req dto.CreateBookingRequest, userID, email string) (paymentDto.CreatePaymentResponse, error)
	GetBookingByID(ctx context.Context, id string) (dto.BookingResponse, error)
	GetUserBookings(ctx context.Context, userID string, req gdto.PaginationRequest) (dto.GetBookingsResponse, error)
	CountUserBookings(ctx context.Context, userID string, req gdto.PaginationRequest) (int, error)
	GetBookedSlots(ctx context.Context, req dto.GetBookedSlotsRequest) (dto.GetBookedSlotsResponse, error)
	CancelUserBooking(ctx context.Context, req dto.CancelUserBookingRequest) error
}

type bookingService struct {
	db             postgres.PgxIface
	repo           repository.Querier
	fieldRepo      fieldRepo.Querier
	paymentService service.PaymentService
	cache          redis.IRedisCache
	cfg            *config.Config
	logger         logger.Interface
}

func New(db postgres.PgxIface, r repository.Querier, f fieldRepo.Querier, p service.PaymentService, c redis.IRedisCache, cfg *config.Config, l logger.Interface) BookingService {
	return &bookingService{
		db:             db,
		repo:           r,
		fieldRepo:      f,
		paymentService: p,
		cache:          c,
		cfg:            cfg,
		logger:         l,
	}
}

const (
	cacheGetBookingKey    = "booking"
	cacheCountBookingsKey = "bookings:count"
	cacheGetBookingsKey   = "bookings"

	identifier = "service - booking - %s"
)

func (s *bookingService) CreateBooking(ctx context.Context, req dto.CreateBookingRequest, userID, email string) (res paymentDto.CreatePaymentResponse, err error) {
	isValid, err := helper.IsBookingTimeValid(req.Date, req.StartTime)
	if err != nil {
		s.logger.Error(identifier, "error validating booking time: "+err.Error())

		return res, failure.BadRequestFromString("invalid booking time format")
	}

	if !isValid {
		s.logger.Error(identifier, "booking time is in the past")

		return res, failure.BadRequestFromString("booking time cannot be in the past")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Error(identifier, "error starting transaction: "+err.Error())

		return res, err
	}

	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.logger.Error(identifier, "error rolling back transaction: "+err.Error())
		}
	}(tx, ctx)

	fieldID := helper.PgUUID(req.FieldID.String())

	startTime, err := helper.PgTimeFromString(req.StartTime)
	if err != nil {
		s.logger.Error(identifier, "error parsing start time: "+err.Error())

		return res, failure.BadRequestFromString("invalid start time format")
	}

	parsedStartTime := helper.TimeFromString(req.StartTime)
	endTimeObj := helper.CalculateEndTime(parsedStartTime, req.Duration)
	endTime := helper.PgTimeFromTime(endTimeObj)

	overlaps, err := s.repo.CountOverlaps(ctx, tx, repository.CountOverlapsParams{
		FieldID:     fieldID,
		BookingDate: helper.PgDate(req.Date),
		Column3:     startTime,
		Column4:     endTime,
	})
	if err != nil {
		s.logger.Error(identifier, "error checking booking overlaps: "+err.Error())

		return res, err
	}

	if overlaps > 0 {
		s.logger.Error(identifier, "booking overlaps with existing bookings")

		return res, failure.Conflict("there are already bookings for this field at this time")
	}

	field, err := s.fieldRepo.GetFieldById(ctx, tx, fieldID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logger.Error(identifier, "field not found with ID: "+fieldID.String())

			return res, failure.NotFound("field not found")
		}

		s.logger.Error(identifier, "error getting field by ID: "+err.Error())

		return res, err
	}

	totalPrice := helper.CalculateTotalPrice(helper.Int64FromPg(field.Price), req.Duration)

	booking, err := s.repo.InsertBooking(ctx, tx, repository.InsertBookingParams{
		UserID:      helper.PgUUID(userID),
		FieldID:     field.ID,
		BookingDate: helper.PgDate(req.Date),
		StartTime:   startTime,
		EndTime:     endTime,
		TotalPrice:  helper.PgInt64(totalPrice),
	})
	if err != nil {
		s.logger.Debug(startTime, endTime)
		s.logger.Error(identifier, "error inserting booking: "+err.Error())

		return res, err
	}

	if err = tx.Commit(ctx); err != nil {
		s.logger.Error(identifier, "error committing transaction: "+err.Error())

		return res, err
	}

	res, err = s.paymentService.CreateInvoice(ctx, paymentDto.CreatePaymentRequest{
		OrderID:    booking.String(),
		Amount:     totalPrice,
		PayerEmail: email,
	})

	go func() {
		ctx := context.WithoutCancel(ctx)

		if err := s.cache.Clear(ctx, helper.BuildCacheKey(cacheGetBookingsKey, "*")); err != nil {
			s.logger.Error(identifier, "error clearing bookings cache: "+err.Error())
		}

		if err := s.cache.Clear(ctx, helper.BuildCacheKey(cacheCountBookingsKey, "*")); err != nil {
			s.logger.Error(identifier, "error clearing bookings cache: "+err.Error())
		}
	}()

	return res, nil
}

func (s *bookingService) GetBookingByID(ctx context.Context, id string) (res dto.BookingResponse, err error) {
	bookingID := helper.PgUUID(id)

	cacheKey := helper.BuildCacheKey(cacheGetBookingKey, id)

	if err = s.cache.Get(ctx, cacheKey, &res); err == nil {
		return res, nil
	}

	booking, err := s.repo.GetBookingById(ctx, s.db, bookingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logger.Error(identifier, "booking not found with ID: "+bookingID.String())

			return res, failure.NotFound("booking not found")
		}

		s.logger.Error(identifier, "error getting booking by ID: "+err.Error())

		return res, err
	}

	res = res.FromModel(booking)

	go func() {
		if err := s.cache.Save(context.WithoutCancel(ctx), cacheKey, res, s.cfg.Cache.Duration); err != nil {
			s.logger.Error(identifier, "error saving booking to cache: "+err.Error())
		}
	}()

	return res, nil
}

func (s *bookingService) GetUserBookings(ctx context.Context, userID string, req gdto.PaginationRequest) (res dto.GetBookingsResponse, err error) {
	page, limit := helper.DefaultPagination(req.Page, req.Limit)

	keyArgs := map[string]string{}
	keyArgs["page"] = strconv.Itoa(page)
	keyArgs["limit"] = strconv.Itoa(limit)
	keyArgs["filter"] = req.Filter
	cacheKey := helper.BuildCacheKey(cacheGetBookingsKey, helper.GenerateUniqueKey(keyArgs))

	var cacheRes dto.GetBookingsResponse

	err = s.cache.Get(ctx, cacheKey, &cacheRes)
	if err == nil {
		s.logger.Info(identifier, "get user bookings - cache hit for user: %s", userID)

		return cacheRes, nil
	}

	totalItems, err := s.CountUserBookings(ctx, userID, req)
	if err != nil {
		s.logger.Error(identifier, "get user bookings - error counting user bookings: %w", err)

		return res, err
	}

	s.logger.Debug(totalItems)

	offset := helper.CalculateOffset(page, limit)

	bookings, err := s.repo.GetBookingsByUserId(ctx, s.db, repository.GetBookingsByUserIdParams{
		UserID:  helper.PgUUID(userID),
		Column2: req.Filter,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		s.logger.Error(identifier, "get user bookings - error getting bookings by user ID: %w", err)

		return res, err
	}

	res.FromModel(bookings, totalItems, limit)

	go func() {
		if err := s.cache.Save(context.WithoutCancel(ctx), cacheKey, res, s.cfg.Cache.Duration); err != nil {
			s.logger.Error(identifier, "get user bookings - failed to save user bookings to cache: %w", err)
		}
	}()

	return res, nil
}

func (s *bookingService) CountUserBookings(ctx context.Context, userID string, req gdto.PaginationRequest) (total int, err error) {
	page, limit := helper.DefaultPagination(req.Page, req.Limit)

	keyArgs := map[string]string{}
	keyArgs["page"] = strconv.Itoa(page)
	keyArgs["limit"] = strconv.Itoa(limit)
	keyArgs["filter"] = req.Filter
	cacheKey := helper.BuildCacheKey(cacheCountBookingsKey, helper.GenerateUniqueKey(keyArgs))

	var cacheRes int

	err = s.cache.Get(ctx, cacheKey, &cacheRes)
	if err == nil {
		s.logger.Info(identifier, "count - cache hit for user bookings: %s", cacheKey)

		return cacheRes, nil
	}

	totalItems, err := s.repo.CountBookingsByUserId(ctx, s.db, repository.CountBookingsByUserIdParams{
		UserID:  helper.PgUUID(userID),
		Column2: req.Filter,
	})
	if err != nil {
		s.logger.Error(identifier, "count - error counting user bookings: %s", err.Error())

		return total, err
	}

	total = int(totalItems)

	go func() {
		if err := s.cache.Save(context.WithoutCancel(ctx), cacheKey, total, s.cfg.Cache.Duration); err != nil {
			s.logger.Error(identifier, "count - error saving user bookings count to cache: %s", err.Error())
		}
	}()

	return total, nil
}

func (s *bookingService) GetBookedSlots(ctx context.Context, req dto.GetBookedSlotsRequest) (res dto.GetBookedSlotsResponse, err error) {
	fieldID := helper.PgUUID(req.FieldID)

	keyArgs := map[string]string{}
	keyArgs["field_id"] = fieldID.String()
	keyArgs["date"] = req.Date
	cacheKey := helper.BuildCacheKey(cacheGetBookingsKey, helper.GenerateUniqueKey(keyArgs))

	var cacheRes dto.GetBookedSlotsResponse

	err = s.cache.Get(ctx, cacheKey, &cacheRes)
	if err == nil {
		s.logger.Info(identifier, "get booked slots - cache hit for key: %s", cacheKey)

		return cacheRes, nil
	}

	slots, err := s.repo.GetBookedTimeSlots(ctx, s.db, repository.GetBookedTimeSlotsParams{
		FieldID:     fieldID,
		BookingDate: helper.PgDate(req.Date),
	})
	if err != nil {
		s.logger.Error(identifier, "get booked slots - error getting booked time slots: %s", err.Error())

		return res, failure.InternalError(err)
	}

	res.FromModel(slots, fieldID.String())

	go func() {
		if err := s.cache.Save(context.WithoutCancel(ctx), cacheKey, res, s.cfg.Cache.Duration); err != nil {
			s.logger.Error(identifier, "get booked slots - error saving booked slots to cache: %s", err.Error())
		}
	}()

	return res, nil
}

func (s *bookingService) CancelUserBooking(ctx context.Context, req dto.CancelUserBookingRequest) (err error) {
	err = s.repo.CancelBooking(ctx, s.db, repository.CancelBookingParams{
		ID:         helper.PgUUID(req.BookingID),
		UserID:     helper.PgUUID(req.UserID),
		CanceledBy: helper.PgString(constant.BookingCanceledByUser),
	})
	if err != nil {
		s.logger.Error(identifier, "cancel user booking - error canceling booking: %s", err.Error())

		return failure.InternalError(err)
	}

	go func() {
		ctx := context.WithoutCancel(ctx)

		if err := s.cache.Delete(ctx, helper.BuildCacheKey(cacheGetBookingKey, req.BookingID)); err != nil {
			s.logger.Error(identifier, "cancel user booking - error deleting booking from cache: %s", err.Error())
		}

		if err := s.cache.Clear(ctx, helper.BuildCacheKey(cacheGetBookingsKey, "*")); err != nil {
			s.logger.Error(identifier, "cancel user booking - error clearing bookings cache: %s", err.Error())
		}

		if err := s.cache.Clear(ctx, helper.BuildCacheKey(cacheCountBookingsKey, "*")); err != nil {
			s.logger.Error(identifier, "cancel user booking - error clearing bookings count cache: %s", err.Error())
		}
	}()

	return nil
}
