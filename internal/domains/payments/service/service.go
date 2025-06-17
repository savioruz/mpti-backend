package service

import (
	"context"
	"errors"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/savioruz/goth/config"
	bookingRepository "github.com/savioruz/goth/internal/domains/bookings/repository"
	"github.com/savioruz/goth/internal/domains/payments/dto"
	"github.com/savioruz/goth/internal/domains/payments/repository"
	"github.com/savioruz/goth/pkg/constant"
	"github.com/savioruz/goth/pkg/failure"
	"github.com/savioruz/goth/pkg/helper"
	"github.com/savioruz/goth/pkg/logger"
	"github.com/savioruz/goth/pkg/postgres"
	"github.com/savioruz/goth/pkg/redis"
	"github.com/xendit/xendit-go/v7"
	"github.com/xendit/xendit-go/v7/invoice"
	"time"
)

type PaymentService interface {
	CreateInvoice(ctx context.Context, req dto.CreatePaymentRequest) (dto.CreatePaymentResponse, error)
	Callbacks(ctx context.Context, req dto.PaymentCallbackRequest, token string) error
}

type paymentService struct {
	db          postgres.PgxIface
	repo        repository.Querier
	bookingRepo bookingRepository.Querier
	cache       redis.IRedisCache
	cfg         *config.Config
	logger      logger.Interface
	xendit      *xendit.APIClient
	validator   *validator.Validate
}

func New(db postgres.PgxIface, r repository.Querier, b bookingRepository.Querier, c redis.IRedisCache, cfg *config.Config, l logger.Interface) PaymentService {
	return &paymentService{
		db:          db,
		repo:        r,
		bookingRepo: b,
		cache:       c,
		cfg:         cfg,
		logger:      l,
		xendit:      xendit.NewClient(cfg.Xendit.APIKey),
		validator:   validator.New(),
	}
}

const (
	identifier = "service - payments- %s"
)

func (s *paymentService) CreateInvoice(ctx context.Context, req dto.CreatePaymentRequest) (res dto.CreatePaymentResponse, err error) {
	if err := s.validator.Struct(req); err != nil {
		s.logger.Error(identifier, " - CreateInvoice - validation error: %v", err)

		return res, failure.BadRequestFromString("validation error: " + err.Error())
	}

	createInvoice := *invoice.NewCreateInvoiceRequest(req.OrderID, float64(req.Amount))

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

	model, err := s.repo.InsertPayment(ctx, tx, repository.InsertPaymentParams{
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

	res = dto.CreatePaymentResponse{
		ID:         model.String(),
		OrderID:    req.OrderID,
		Amount:     req.Amount,
		Status:     paymentStatus,
		ExpiryDate: expiryDate,
		PaymentURL: paymentURL,
	}

	if err = tx.Commit(ctx); err != nil {
		s.logger.Error(identifier, " - CreateInvoice - failed to commit transaction: %v", err)

		return res, failure.InternalError(err)
	}

	return res, nil
}

func (s *paymentService) Callbacks(ctx context.Context, req dto.PaymentCallbackRequest, token string) (err error) {
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

	if err := s.repo.UpdatePaymentStatus(ctx, tx, repository.UpdatePaymentStatusParams{
		TransactionID: req.ExternalID,
		PaymentStatus: req.Status,
		PaidAt:        helper.PgTimestamp(time.Now()),
	}); err != nil {
		s.logger.Error(identifier, " - Callbacks - failed to update payment status: %v", err)

		if errors.Is(err, pgx.ErrNoRows) {
			return failure.NotFound("payment not found for transaction ID: " + req.ExternalID)
		}

		return failure.InternalError(err)
	}

	if err = s.bookingRepo.UpdateBookingStatus(ctx, tx, bookingRepository.UpdateBookingStatusParams{
		ID:     helper.PgUUID(req.ExternalID),
		Status: req.Status,
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

	s.logger.Info(identifier, " - Callbacks - payment status updated successfully for transaction ID: %s", req.ExternalID)

	return nil
}
