package helper

import (
	"github.com/savioruz/goth/pkg/constant"
	"time"
)

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

func CalculateEndTime(startTime time.Time, durationMinutes int) time.Time {
	return startTime.Add(time.Duration(durationMinutes) * time.Minute)
}

func CalculateTotalPrice(pricePerHour int64, durationMinutes int) int64 {
	if pricePerHour <= 0 || durationMinutes <= 0 {
		return 0
	}

	hours := durationMinutes / constant.SecondsPerMinute
	minutes := durationMinutes % constant.SecondsPerMinute

	totalPrice := pricePerHour * int64(hours)
	if minutes > 0 {
		totalPrice += pricePerHour * int64(minutes) / constant.SecondsPerMinute
	}

	return totalPrice
}
