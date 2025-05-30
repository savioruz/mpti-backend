package constant

import "time"

const (
	CacheParentKey = "mpti-backend"
)

const (
	DateFormat = time.RFC3339
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
