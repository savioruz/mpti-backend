package dto

import (
	"github.com/savioruz/goth/internal/domains/bookings/repository"
	"github.com/savioruz/goth/pkg/constant"
	"github.com/savioruz/goth/pkg/helper"
)

type BookingResponse struct {
	ID          string `json:"id"`
	FieldID     string `json:"field_id"`
	FieldName   string `json:"field_name,omitempty"`
	BookingDate string `json:"booking_date"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	TotalPrice  int64  `json:"total_price"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func (b BookingResponse) FromModel(model repository.Booking) BookingResponse {
	startTime, _ := helper.PgTimeToString(model.StartTime)
	endTime, _ := helper.PgTimeToString(model.EndTime)

	return BookingResponse{
		ID:          model.ID.String(),
		FieldID:     model.FieldID.String(),
		BookingDate: model.BookingDate.Time.Format(constant.DateFormat),
		StartTime:   startTime,
		EndTime:     endTime,
		TotalPrice:  helper.Int64FromPg(model.TotalPrice),
		Status:      model.Status,
		CreatedAt:   model.CreatedAt.Time.Format(constant.FullDateFormat),
		UpdatedAt:   model.UpdatedAt.Time.Format(constant.FullDateFormat),
	}
}

type GetBookingsResponse struct {
	Bookings   []BookingResponse `json:"bookings"`
	TotalItems int               `json:"total_items"`
	TotalPages int               `json:"total_pages"`
}

func (b *GetBookingsResponse) FromModel(bookings []repository.Booking, totalItems, limit int) {
	b.TotalItems = totalItems
	b.TotalPages = helper.CalculateTotalPages(totalItems, limit)

	if len(bookings) == 0 {
		b.Bookings = []BookingResponse{}

		return
	}

	b.Bookings = make([]BookingResponse, len(bookings))

	for i, booking := range bookings {
		b.Bookings[i] = BookingResponse{}.FromModel(booking)
	}
}

// EnrichWithFieldNames adds field names to the booking responses
func (b *GetBookingsResponse) EnrichWithFieldNames(fieldNames map[string]string) {
	for i := range b.Bookings {
		if name, exists := fieldNames[b.Bookings[i].FieldID]; exists {
			b.Bookings[i].FieldName = name
		}
	}
}

type BookedSlot struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type GetBookedSlotsResponse struct {
	FieldID     string       `json:"field_id"`
	BookedSlots []BookedSlot `json:"booked_slots"`
	TotalItems  int          `json:"total_items"`
}

func (b *GetBookedSlotsResponse) FromModel(bookedSlots []repository.GetBookedTimeSlotsRow, fieldID string) {
	b.FieldID = fieldID

	if len(bookedSlots) == 0 {
		b.BookedSlots = []BookedSlot{}
		b.TotalItems = 0

		return
	}

	b.BookedSlots = make([]BookedSlot, len(bookedSlots))
	b.TotalItems = len(bookedSlots)

	for i, slot := range bookedSlots {
		startTime, _ := helper.PgTimeToString(slot.StartTime)
		endTime, _ := helper.PgTimeToString(slot.EndTime)

		b.BookedSlots[i] = BookedSlot{
			StartTime: startTime,
			EndTime:   endTime,
		}
	}
}
