package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/savioruz/goth/config"
	bookingRepository "github.com/savioruz/goth/internal/domains/bookings/repository"
	"github.com/savioruz/goth/internal/domains/payments/dto"
	"github.com/savioruz/goth/internal/domains/payments/repository"
	userRepository "github.com/savioruz/goth/internal/domains/user/repository"
	"github.com/savioruz/goth/pkg/constant"
	"github.com/savioruz/goth/pkg/failure"
	"github.com/savioruz/goth/pkg/helper"
	"github.com/savioruz/goth/pkg/logger"
	"github.com/savioruz/goth/pkg/mail"
	"github.com/savioruz/goth/pkg/postgres"
	"github.com/savioruz/goth/pkg/redis"
	"github.com/xendit/xendit-go/v7"
	"github.com/xendit/xendit-go/v7/invoice"
)

type PaymentService interface {
	CreateInvoice(ctx context.Context, req dto.CreatePaymentInvoice) (dto.CreatePaymentInvoiceResponse, error)
	Callbacks(ctx context.Context, req dto.CallbackPaymentInvoice, token string) error
	CreatePayments(ctx context.Context, req dto.CreatePaymentRequest) (string, error)
	GetPayments(ctx context.Context, req dto.GetPaymentsRequest) (dto.PaginatedPaymentResponse, error)
	GetPaymentsByBookingID(ctx context.Context, bookingID string) ([]dto.PaymentResponse, error)
}

type paymentService struct {
	db          postgres.PgxIface
	repo        repository.Querier
	bookingRepo bookingRepository.Querier
	userRepo    userRepository.Querier
	cache       redis.IRedisCache
	cfg         *config.Config
	logger      logger.Interface
	xendit      *xendit.APIClient
	validator   *validator.Validate
	mailService mail.Service
}

func New(db postgres.PgxIface, r repository.Querier, b bookingRepository.Querier, u userRepository.Querier, c redis.IRedisCache, cfg *config.Config, l logger.Interface, m mail.Service) PaymentService {
	return &paymentService{
		db:          db,
		repo:        r,
		bookingRepo: b,
		userRepo:    u,
		cache:       c,
		cfg:         cfg,
		logger:      l,
		xendit:      xendit.NewClient(cfg.Xendit.APIKey),
		validator:   validator.New(),
		mailService: m,
	}
}

const (
	identifier = "service - payments- %s"
)

func (s *paymentService) CreateInvoice(ctx context.Context, req dto.CreatePaymentInvoice) (res dto.CreatePaymentInvoiceResponse, err error) {
	if err := s.validator.Struct(req); err != nil {
		s.logger.Error(identifier, " - CreateInvoice - validation error: %v", err)

		return res, failure.BadRequestFromString("validation error: " + err.Error())
	}

	createInvoice := *invoice.NewCreateInvoiceRequest(req.OrderID, float64(req.Amount))
	createInvoice.SuccessRedirectUrl = &s.cfg.Xendit.SuccessURL
	createInvoice.FailureRedirectUrl = &s.cfg.Xendit.FailureURL

	invoiceResult, _, erro := s.xendit.InvoiceApi.CreateInvoice(ctx).CreateInvoiceRequest(createInvoice).Execute()
	if erro != nil {
		s.logger.Error(identifier, " - CreateInvoice - failed to create invoiceResult: %v", erro)

		return res, failure.InternalError(erro)
	}

	// Check for nil fields in invoiceResult
	paymentMethod := "UNKNOWN"
	if invoiceResult.PaymentMethod != nil {
		paymentMethod = invoiceResult.PaymentMethod.String()
	}

	paymentStatus := "UNKNOWN"
	if invoiceResult.Status != "" {
		paymentStatus = invoiceResult.Status.String()
	}

	transactionID := ""
	if invoiceResult.Id != nil {
		transactionID = *invoiceResult.Id
	} else {
		s.logger.Error(identifier, " - CreateInvoice - invoice ID is nil")

		return res, failure.InternalError(err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Error(identifier, " - CreateInvoice - failed to begin transaction: %v", err)

		return res, failure.InternalError(err)
	}

	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.logger.Error(identifier, " - CreateInvoice - failed to rollback transaction: %v", err)
		}
	}(tx, ctx)

	id, err := s.repo.InsertPayment(ctx, tx, repository.InsertPaymentParams{
		BookingID:     helper.PgUUID(req.OrderID),
		PaymentMethod: paymentMethod,
		PaymentStatus: paymentStatus,
		TransactionID: transactionID,
	})
	if err != nil {
		s.logger.Error(identifier, " - CreateInvoice - failed to insert payment: %v", err)

		return res, failure.InternalError(err)
	}

	expiryDate := invoiceResult.ExpiryDate.Format(constant.DateFormat)

	paymentURL := ""
	if invoiceResult.InvoiceUrl != "" {
		paymentURL = invoiceResult.InvoiceUrl
	}

	res = dto.CreatePaymentInvoiceResponse{
		ID:         id.String(),
		OrderID:    req.OrderID,
		Amount:     req.Amount,
		Status:     paymentStatus,
		ExpiryDate: &expiryDate,
		PaymentURL: &paymentURL,
	}

	if err = tx.Commit(ctx); err != nil {
		s.logger.Error(identifier, " - CreateInvoice - failed to commit transaction: %v", err)

		return res, failure.InternalError(err)
	}

	return res, nil
}

func (s *paymentService) Callbacks(ctx context.Context, req dto.CallbackPaymentInvoice, token string) (err error) {
	if s.cfg.Xendit.CallbackToken != token {
		s.logger.Error(identifier, " - Callbacks - invalid callback token: %s", token)

		return failure.Unauthorized("invalid callback token")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Error(identifier, " - Callbacks - failed to begin transaction: %v", err)

		return failure.InternalError(err)
	}

	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.logger.Error(identifier, " - Callbacks - failed to rollback transaction: %v", err)
		}
	}(tx, ctx)

	paymentStatus := req.Status
	if req.Status == "" {
		paymentStatus = constant.PaymentStatusPending
	}

	paymentMethod := req.PaymentMethod
	if paymentMethod == nil {
		paymentMethod = &constant.PaymentUnknownMethod
	}

	if err := s.repo.UpdatePaymentStatusByBookingID(ctx, tx, repository.UpdatePaymentStatusByBookingIDParams{
		BookingID:     helper.PgUUID(req.ExternalID),
		PaymentStatus: paymentStatus,
		PaymentMethod: *paymentMethod,
		PaidAt:        helper.PgTimestampNow(),
	}); err != nil {
		s.logger.Error(identifier, " - Callbacks - failed to update payment status: %v", err)

		if errors.Is(err, pgx.ErrNoRows) {
			return failure.NotFound("payment not found for booking ID: " + req.ExternalID)
		}

		return failure.InternalError(err)
	}

	var bookingStatus string
	if req.Status == constant.PaymentStatusPaid {
		bookingStatus = constant.BookingStatusConfirmed
	}

	if err = s.bookingRepo.UpdateBookingStatus(ctx, tx, bookingRepository.UpdateBookingStatusParams{
		ID:     helper.PgUUID(req.ExternalID),
		Status: bookingStatus,
	}); err != nil {
		s.logger.Error(identifier, " - Callbacks - failed to update booking status: %v", err)

		if errors.Is(err, pgx.ErrNoRows) {
			return failure.NotFound("booking not found for transaction ID: " + req.ExternalID)
		}

		return failure.InternalError(err)
	}

	if err = tx.Commit(ctx); err != nil {
		s.logger.Error(identifier, " - Callbacks - failed to commit transaction: %v", err)

		return failure.InternalError(err)
	}

	s.logger.Info(identifier, " - Callbacks - payment status updated successfully for booking ID: %s", req.ExternalID)

	// Send booking confirmation email if payment is successful
	if req.Status == constant.PaymentStatusPaid {
		go func() {
			if err := s.sendBookingConfirmationEmail(ctx, req.ExternalID, *paymentMethod); err != nil {
				s.logger.Error(identifier, " - Callbacks - failed to send booking confirmation email: %v", err)
			}
		}()
	}

	return nil
}

func (s *paymentService) CreatePayments(ctx context.Context, req dto.CreatePaymentRequest) (id string, err error) {
	if err := s.validator.Struct(req); err != nil {
		s.logger.Error(identifier, " - CreatePayments - validation error: %v", err)

		return "", failure.BadRequestFromString("validation error: " + err.Error())
	}

	var paymentStatus string
	if req.PaymentMethod == constant.PaymentCashMethod {
		paymentStatus = constant.PaymentStatusPaid
	} else {
		paymentStatus = constant.PaymentStatusPending
	}

	res, err := s.repo.InsertPayment(ctx, s.db, repository.InsertPaymentParams{
		BookingID:     helper.PgUUID(req.BookingID),
		PaymentMethod: req.PaymentMethod,
		PaymentStatus: paymentStatus,
		TransactionID: req.TransactionID,
	})
	if err != nil {
		s.logger.Error(identifier, " - CreatePayments - failed to create payment: %v", err)

		return "", failure.InternalError(err)
	}

	return res.String(), nil
}

func (s *paymentService) GetPayments(ctx context.Context, req dto.GetPaymentsRequest) (res dto.PaginatedPaymentResponse, err error) {
	if err := s.validator.Struct(req); err != nil {
		s.logger.Error(identifier, " - GetPayments - validation error: %v", err)

		return res, failure.BadRequestFromString("validation error: " + err.Error())
	}

	// Set default pagination values
	page := req.Page
	if page <= 0 {
		page = 1
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	offset := (page - 1) * limit

	totalCount, err := s.repo.CountPayments(ctx, s.db, repository.CountPaymentsParams{
		Column1: req.PaymentMethod,
		Column2: req.PaymentStatus,
	})
	if err != nil {
		s.logger.Error(identifier, " - GetPayments - failed to count payments: %v", err)

		return res, failure.InternalError(err)
	}

	// Get paginated payments
	payments, err := s.repo.GetPayments(ctx, s.db, repository.GetPaymentsParams{
		Column1: req.PaymentMethod,
		Column2: req.PaymentStatus,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		s.logger.Error(identifier, " - GetPayments - failed to get payments: %v", err)

		return res, failure.InternalError(err)
	}

	res.FromModel(payments, int(totalCount), limit)

	return res, nil
}

func (s *paymentService) GetPaymentsByBookingID(ctx context.Context, bookingID string) (res []dto.PaymentResponse, err error) {
	if bookingID == "" {
		return res, failure.BadRequestFromString("booking ID is required")
	}

	payments, err := s.repo.GetPaymentsByBookingID(ctx, s.db, helper.PgUUID(bookingID))
	if err != nil {
		s.logger.Error(identifier, " - GetPaymentsByBookingID - failed to get payments: %v", err)

		return res, failure.InternalError(err)
	}

	paymentResponses := make([]dto.PaymentResponse, len(payments))
	for i, payment := range payments {
		paymentResponses[i] = dto.PaymentResponse{}.FromModel(payment)
	}

	return paymentResponses, nil
}

// sendBookingConfirmationEmail sends confirmation email after successful payment
func (s *paymentService) sendBookingConfirmationEmail(ctx context.Context, bookingID, paymentMethod string) error {
	// Get booking details
	booking, err := s.bookingRepo.GetBookingById(ctx, s.db, helper.PgUUID(bookingID))
	if err != nil {
		return fmt.Errorf("failed to get booking details: %w", err)
	}

	// Get user details
	user, err := s.userRepo.GetUserByID(ctx, s.db, booking.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user details: %w", err)
	}

	// Prepare email data
	startTime, _ := helper.PgTimeToString(booking.StartTime)
	endTime, _ := helper.PgTimeToString(booking.EndTime)

	emailData := mail.BookingConfirmationData{
		CustomerName:     user.FullName.String,
		BookingID:        bookingID,
		Status:           constant.BookingStatusConfirmed,
		BookingDate:      booking.BookingDate.Time.Format("2006-01-02"),
		StartTime:        startTime,
		EndTime:          endTime,
		TotalAmount:      helper.FormatAmountFromCents(booking.TotalPrice.Int.Int64()),
		PaymentMethod:    paymentMethod,
		ConfirmationDate: time.Now().Format("2006-01-02 15:04:05"),
	}

	// Send email
	return s.mailService.SendBookingConfirmationEmail(user.Email, emailData)
}
