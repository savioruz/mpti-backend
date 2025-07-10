package helper

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/savioruz/goth/pkg/constant"
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

	timezone := time.UTC
	if AppTimezone != nil {
		timezone = AppTimezone
	}

	bookingDateTime := time.Date(
		bookingDateObj.Year(), bookingDateObj.Month(), bookingDateObj.Day(),
		startTime.Hour(), startTime.Minute(), 0, 0, timezone,
	)

	return bookingDateTime.After(NowInAppTimezone()), nil
}

func GenerateStateToken() string {
	const length = 16
	b := make([]byte, length)

	_, err := rand.Read(b)
	if err != nil {
		return NowInAppTimezone().String()
	}

	return base64.URLEncoding.EncodeToString(b)
}

// IsValidImageType checks if the content type is a valid image type
func IsValidImageType(contentType string) bool {
	validTypes := []string{
		constant.ContentTypeJPEG,
		constant.ContentTypeJPG,
		constant.ContentTypePNG,
		constant.ContentTypeGIF,
		constant.ContentTypeWEBP,
	}

	for _, validType := range validTypes {
		if contentType == validType {
			return true
		}
	}

	return false
}

// IsAllowedEmailDomain checks if the email domain is from an allowed provider
func IsAllowedEmailDomain(email string) bool {
	allowedDomains := []string{
		"gmail.com",
		"googlemail.com",
		"outlook.com",
		"hotmail.com",
		"live.com",
		"yahoo.com",
		"yahoo.co.uk",
		"yahoo.co.id",
		"yahoo.co.jp",
		"icloud.com",
	}

	partLen := 2
	parts := strings.Split(email, "@")

	if len(parts) != partLen {
		return false
	}

	domain := strings.ToLower(parts[1])

	for _, allowedDomain := range allowedDomains {
		if domain == allowedDomain {
			return true
		}
	}

	return false
}
