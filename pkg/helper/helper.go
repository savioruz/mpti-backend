package helper

import (
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/savioruz/goth/pkg/constant"
	"math/big"
	"sort"
)

const (
	x = 10
)

// PgString converts a string to pgtype.Text
func PgString(s string) pgtype.Text {
	return pgtype.Text{
		String: s,
		Valid:  true,
	}
}

// PgInt64 converts an int64 to pgtype.Numeric
func PgInt64(i int64) pgtype.Numeric {
	bigInt := new(big.Int).SetInt64(i)

	return pgtype.Numeric{
		Int:   bigInt,
		Valid: true,
	}
}

// Int64FromPg converts a pgtype.Numeric to an int64
func Int64FromPg(n pgtype.Numeric) int64 {
	if !n.Valid || n.Int == nil {
		return 0
	}

	if n.Exp != 0 {
		// Create a copy of the number to work with
		result := new(big.Int).Set(n.Int)

		// If Exp is negative, divide by 10^(-Exp)
		if n.Exp < 0 {
			divisor := new(big.Int).Exp(big.NewInt(x), big.NewInt(int64(-n.Exp)), nil)
			result = result.Div(result, divisor)
		} else {
			// If Exp is positive, multiply by 10^Exp
			multiplier := new(big.Int).Exp(big.NewInt(x), big.NewInt(int64(n.Exp)), nil)
			result = result.Mul(result, multiplier)
		}

		return result.Int64()
	}

	return n.Int.Int64()
}

// PgUUID converts a string UUID to pgtype.UUID
func PgUUID(id string) pgtype.UUID {
	var uuid pgtype.UUID

	err := uuid.Scan(id)
	if err != nil {
		return pgtype.UUID{Valid: false}
	}

	return uuid
}

// GenerateUniqueKey generates a unique key based on the provided map
func GenerateUniqueKey(args map[string]string) string {
	var keys []string
	for k := range args {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var uniqueKey string
	for _, k := range keys {
		uniqueKey += fmt.Sprintf("%s=%s;", k, args[k])
	}

	return uniqueKey
}

// BuildCacheKey builds a cache key based on the provided key and optional postfix
func BuildCacheKey(key string, postfix ...string) string {
	if len(postfix) > 0 && postfix[0] != "" {
		return fmt.Sprintf("%s:cache:%s:%s", constant.CacheParentKey, key, postfix[0])
	}

	return fmt.Sprintf("%s:cache:%s", constant.CacheParentKey, key)
}

func DefaultPagination(page, limit int) (resultPage, resultLimit int) {
	resultPage = page
	if resultPage <= 0 {
		resultPage = constant.PaginationDefaultPage
	}

	resultLimit = limit
	if resultLimit <= 0 {
		resultLimit = constant.PaginationDefaultLimit
	}

	return resultPage, resultLimit
}

func CalculateOffset(page, limit int) int {
	if page <= 0 || limit <= 0 {
		return 0
	}

	return (page - 1) * limit
}

func CalculateTotalPages(totalItems, limit int) int {
	if totalItems <= 0 || limit <= 0 {
		return 1
	}

	return (totalItems + limit - 1) / limit
}
