package constant

import (
	"errors"
	"time"
)

const (
	CacheParentKey = "mpti-backend"
)

const (
	RequestParamID = "id"

	RequestValidateUUID = "required,uuid"
)

const (
	BookingStatusPending   = "PENDING"
	BookingStatusCanceled  = "CANCELLED"
	BookingStatusExpired   = "EXPIRED"
	BookingStatusPaid      = "PAID"
	BookingStatusConfirmed = "CONFIRMED"

	BookingCanceledByUser   = "user"
	BookingCanceledByAdmin  = "admin"
	BookingCanceledBySystem = "system"
)

var PaymentUnknownMethod = "UNKNOWN"

const (
	PaymentCurrencyIDR = "IDR"

	PaymentCashMethod = "CASH"

	PaymentStatusPaid    = "PAID"
	PaymentStatusPending = "PENDING"
)

const (
	RequestHeaderCallback = "x-callback-token"
)

const (
	FullDateFormat  = time.RFC3339
	DateFormat      = "2006-01-02"
	HoursFormat     = "15:04"
	TimestampFormat = "2006-01-02 15:04:05"

	SecondsPerHour     = 3600
	MinutesPerHour     = 60
	MicrosecondsPerSec = 1000000
)

const (
	UserRoleAdmin = "9"
	UserRoleStaff = "2"
	UserRoleUser  = "1"
)

const (
	JwtFieldUser  = "user_id"
	JwtFieldEmail = "email"
	JwtFieldLevel = "level"
)

const (
	PaginationDefaultLimit = 10
	PaginationDefaultPage  = 1
)

const (
	// File upload constants
	MaxUploadSize = 5 << 20 // 50MB total

	// Valid image content types
	ContentTypeJPEG = "image/jpeg"
	ContentTypeJPG  = "image/jpg"
	ContentTypePNG  = "image/png"
	ContentTypeGIF  = "image/gif"
	ContentTypeWEBP = "image/webp"
)

var (
	ErrInvalidContextUserType = errors.New("invalid user type in context")
)
