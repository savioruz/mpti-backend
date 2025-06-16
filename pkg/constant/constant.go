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
	BookingStatusPending   = "pending"
	BookingStatusCanceled  = "canceled"
	BookingStatusExpired   = "expired"
	BookingStatusConfirmed = "confirmed"

	BookingCanceledByUser   = "user"
	BookingCanceledByAdmin  = "admin"
	BookingCanceledBySystem = "system"
)

const (
	PaymentCurrencyIDR = "IDR"
)

const (
	RequestHeaderCallback = "x-callback-token"
)

const (
	FullDateFormat = time.RFC3339
	DateFormat     = "2006-01-02"
	HoursFormat    = "15:04"

	SecondsPerHour     = 3600
	MinutesPerHour     = 60
	MicrosecondsPerSec = 1000000
)

const (
	UserRoleAdmin = "9"
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

var (
	ErrInvalidContextUserType = errors.New("invalid user type in context")
)
