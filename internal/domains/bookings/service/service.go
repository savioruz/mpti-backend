package service

import (
	"context"
	"errors"
	"strconv"

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
)

type BookingService interface {
	CreateBooking(ctx context.Context, req dto.CreateBookingRequest, userID, email, userRole string) (paymentDto.CreatePaymentInvoiceResponse, error)
	GetBookingByID(ctx context.Context, id string) (dto.BookingResponse, error)
	GetUserBookings(ctx context.Context, userID string, req gdto.PaginationRequest) (dto.GetBookingsResponse, error)
	CountUserBookings(ctx context.Context, userID string, req gdto.PaginationRequest) (int, error)
	GetAllBookings(ctx context.Context, req gdto.PaginationRequest) (dto.GetBookingsResponse, error)
	CountAllBookings(ctx context.Context, req gdto.PaginationRequest) (int, error)
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

func (s *bookingService) CreateBooking(ctx context.Context, req dto.CreateBookingRequest, userID, email, userRole string) (res paymentDto.CreatePaymentInvoiceResponse, err error) {
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

	var status string
	if *req.Cash {
		status = constant.BookingStatusPaid
	} else {
		status = constant.BookingStatusPending
	}

	totalPrice := helper.CalculateTotalPrice(helper.Int64FromPg(field.Price), req.Duration)

	booking, err := s.repo.InsertBooking(ctx, tx, repository.InsertBookingParams{
		UserID:      helper.PgUUID(userID),
		FieldID:     field.ID,
		BookingDate: helper.PgDate(req.Date),
		StartTime:   startTime,
		EndTime:     endTime,
		TotalPrice:  helper.PgInt64(totalPrice),
		Status:      status,
	})
	if err != nil {
		s.logger.Error(identifier, "error inserting booking: "+err.Error())

		return res, err
	}

	if err = tx.Commit(ctx); err != nil {
		s.logger.Error(identifier, "error committing transaction: "+err.Error())

		return res, err
	}

	if *req.Cash {
		// Check if user has permission to create cash payments (staff or admin only)
		if userRole != constant.UserRoleAdmin && userRole != constant.UserRoleStaff {
			s.logger.Error(identifier, "unauthorized cash payment attempt by user role: %s", userRole)

			return res, failure.Forbidden("only staff and admin can create cash payments")
		}

		transactionID := "cash-" + booking.String()

		id, err := s.paymentService.CreatePayments(ctx, paymentDto.CreatePaymentRequest{
			BookingID:     booking.String(),
			PaymentMethod: constant.PaymentCashMethod,
			TransactionID: transactionID,
			Amount:        totalPrice,
		})
		if err != nil {
			s.logger.Error(identifier, "error creating cash payment: "+err.Error())

			return res, err
		}

		res = paymentDto.CreatePaymentInvoiceResponse{
			ID:         id,
			OrderID:    booking.String(),
			Amount:     totalPrice,
			Status:     constant.PaymentStatusPaid,
			ExpiryDate: nil,
			PaymentURL: nil,
		}
	} else {
		res, err = s.paymentService.CreateInvoice(ctx, paymentDto.CreatePaymentInvoice{
			OrderID:    booking.String(),
			Amount:     totalPrice,
			PayerEmail: email,
		})
		if err != nil {
			s.logger.Error(identifier, "error creating payment invoice: "+err.Error())

			return res, err
		}
	}

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

	// Get field name
	field, err := s.fieldRepo.GetFieldById(ctx, s.db, booking.FieldID)
	if err == nil {
		res.FieldName = field.Name
	} else {
		s.logger.Error(identifier, "error getting field name for ID %s: %w", booking.FieldID.String(), err)
	}

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

	// Create the basic response from bookings
	res.FromModel(bookings, totalItems, limit)

	// Collect all field IDs
	fieldIDs := make(map[string]struct{})

	for _, booking := range bookings {
		fieldID := booking.FieldID.String()
		fieldIDs[fieldID] = struct{}{}
	}

	// Get field names for all field IDs
	fieldNames := make(map[string]string)

	for fieldID := range fieldIDs {
		field, err := s.fieldRepo.GetFieldById(ctx, s.db, helper.PgUUID(fieldID))
		if err == nil {
			fieldNames[fieldID] = field.Name
		} else {
			s.logger.Error(identifier, "get user bookings - error getting field name for ID %s: %w", fieldID, err)
		}
	}

	// Enrich the response with field names
	res.EnrichWithFieldNames(fieldNames)

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

func (s *bookingService) GetAllBookings(ctx context.Context, req gdto.PaginationRequest) (res dto.GetBookingsResponse, err error) {
	page, limit := helper.DefaultPagination(req.Page, req.Limit)

	keyArgs := map[string]string{}
	keyArgs["page"] = strconv.Itoa(page)
	keyArgs["limit"] = strconv.Itoa(limit)
	keyArgs["filter"] = req.Filter
	cacheKey := helper.BuildCacheKey(cacheGetBookingsKey, "all:"+helper.GenerateUniqueKey(keyArgs))

	var cacheRes dto.GetBookingsResponse

	err = s.cache.Get(ctx, cacheKey, &cacheRes)
	if err == nil {
		s.logger.Info(identifier, "get all bookings - cache hit for key: %s", cacheKey)

		return cacheRes, nil
	}

	totalItems, err := s.CountAllBookings(ctx, req)
	if err != nil {
		s.logger.Error(identifier, "get all bookings - error counting all bookings: %w", err)

		return res, err
	}

	offset := helper.CalculateOffset(page, limit)

	bookings, err := s.repo.GetAllBookings(ctx, s.db, repository.GetAllBookingsParams{
		Column1: req.Filter,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		s.logger.Error(identifier, "get all bookings - error getting all bookings: %w", err)

		return res, err
	}

	// Create the basic response from bookings
	res.FromModel(bookings, totalItems, limit)

	// Collect all field IDs
	fieldIDs := make(map[string]struct{})

	for _, booking := range bookings {
		fieldID := booking.FieldID.String()
		fieldIDs[fieldID] = struct{}{}
	}

	// Get field names for all field IDs
	fieldNames := make(map[string]string)

	for fieldID := range fieldIDs {
		field, err := s.fieldRepo.GetFieldById(ctx, s.db, helper.PgUUID(fieldID))
		if err == nil {
			fieldNames[fieldID] = field.Name
		} else {
			s.logger.Error(identifier, "get all bookings - error getting field name for ID %s: %w", fieldID, err)
		}
	}

	// Enrich the response with field names
	res.EnrichWithFieldNames(fieldNames)

	go func() {
		if err := s.cache.Save(context.WithoutCancel(ctx), cacheKey, res, s.cfg.Cache.Duration); err != nil {
			s.logger.Error(identifier, "get all bookings - failed to save all bookings to cache: %w", err)
		}
	}()

	return res, nil
}

func (s *bookingService) CountAllBookings(ctx context.Context, req gdto.PaginationRequest) (total int, err error) {
	page, limit := helper.DefaultPagination(req.Page, req.Limit)

	keyArgs := map[string]string{}
	keyArgs["page"] = strconv.Itoa(page)
	keyArgs["limit"] = strconv.Itoa(limit)
	keyArgs["filter"] = req.Filter
	cacheKey := helper.BuildCacheKey(cacheCountBookingsKey, "all:"+helper.GenerateUniqueKey(keyArgs))

	var cacheRes int

	err = s.cache.Get(ctx, cacheKey, &cacheRes)
	if err == nil {
		s.logger.Info(identifier, "count all bookings - cache hit for key: %s", cacheKey)

		return cacheRes, nil
	}

	totalItems, err := s.repo.CountAllBookings(ctx, s.db, req.Filter)
	if err != nil {
		s.logger.Error(identifier, "count all bookings - error counting all bookings: %s", err.Error())

		return total, err
	}

	total = int(totalItems)

	go func() {
		if err := s.cache.Save(context.WithoutCancel(ctx), cacheKey, total, s.cfg.Cache.Duration); err != nil {
			s.logger.Error(identifier, "count all bookings - error saving all bookings count to cache: %s", err.Error())
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
