package dto

import (
	"github.com/savioruz/goth/internal/domains/locations/repository"
	"github.com/savioruz/goth/pkg/constant"
	"github.com/savioruz/goth/pkg/helper"
)

type LocationResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Description string  `json:"description,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func (l LocationResponse) FromModel(model repository.Location) LocationResponse {
	return LocationResponse{
		ID:          model.ID.String(),
		Name:        model.Name,
		Latitude:    model.Latitude,
		Longitude:   model.Longitude,
		Description: model.Description.String,
		CreatedAt:   model.CreatedAt.Time.Format(constant.FullDateFormat),
		UpdatedAt:   model.UpdatedAt.Time.Format(constant.FullDateFormat),
	}
}

type PaginatedLocationResponse struct {
	Locations  []LocationResponse `json:"locations"`
	TotalItems int                `json:"total_items"`
	TotalPages int                `json:"total_pages"`
}

func (l *PaginatedLocationResponse) FromModel(locations []repository.Location, totalItems, limit int) {
	l.TotalItems = totalItems
	l.TotalPages = helper.CalculateTotalPages(totalItems, limit)

	if len(locations) == 0 {
		l.Locations = []LocationResponse{}

		return
	}

	l.Locations = make([]LocationResponse, len(locations))

	for i, location := range locations {
		l.Locations[i] = LocationResponse{}.FromModel(location)
	}
}
