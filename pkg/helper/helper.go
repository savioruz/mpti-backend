package helper

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/savioruz/goth/pkg/constant"
	"sort"
	"time"
)

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

// IsBookingTimeValid checks if booking time is not in the past
func IsBookingTimeValid(bookingDate string, startTimeStr string) (bool, error) {
	bookingDateObj, err := time.Parse(constant.DateFormat, bookingDate)
	if err != nil {
		return false, err
	}

	startTime, err := time.Parse(constant.HoursFormat, startTimeStr)
	if err != nil {
		return false, err
	}

	bookingDateTime := time.Date(
		bookingDateObj.Year(), bookingDateObj.Month(), bookingDateObj.Day(),
		startTime.Hour(), startTime.Minute(), 0, 0, time.Local,
	)

	return bookingDateTime.After(time.Now()), nil
}

func GenerateStateToken() string {
	const length = 16
	b := make([]byte, length)

	_, err := rand.Read(b)
	if err != nil {
		return time.Now().String()
	}

	return base64.URLEncoding.EncodeToString(b)
}
