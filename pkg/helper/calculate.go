package helper

import (
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

func CalculateEndTime(startTime time.Time, durationHours int) time.Time {
	return startTime.Add(time.Duration(durationHours) * time.Hour)
}

func CalculateTotalPrice(pricePerHour int64, durationHours int) int64 {
	if pricePerHour <= 0 || durationHours <= 0 {
		return 0
	}

	return pricePerHour * int64(durationHours)
}
