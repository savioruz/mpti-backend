package dto

import (
	"github.com/google/uuid"
)

type CreateBookingRequest struct {
	FieldID   uuid.UUID `json:"field_id" validate:"required,uuid"`
	Date      string    `json:"date" validate:"required,datetime=2006-01-02" example:"2006-01-02"`
	StartTime string    `json:"start_time" validate:"required,datetime=15:04" example:"15:04"`
	Duration  int       `json:"duration" validate:"required"`
	Cash      *bool     `json:"cash" validate:"required"`
}

type GetBookedSlotsRequest struct {
	FieldID string `json:"field_id" validate:"required,uuid"`
	Date    string `json:"date" validate:"required,datetime=2006-01-02" example:"2006-01-02"`
}

type CancelUserBookingRequest struct {
	BookingID string `json:"booking_id" validate:"required,uuid" swaggerignore:"true"`
	UserID    string `json:"user_id" validate:"required,uuid" swaggerignore:"true"`
}
