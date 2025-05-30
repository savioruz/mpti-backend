package dto

import (
	"github.com/savioruz/goth/internal/domains/fields/repository"
	"github.com/savioruz/goth/pkg/constant"
	"github.com/savioruz/goth/pkg/helper"
)

type FieldResponse struct {
	ID          string `json:"id"`
	LocationID  string `json:"location_id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Price       int64  `json:"price"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func (f FieldResponse) FromModel(model repository.Field) FieldResponse {
	return FieldResponse{
		ID:          model.ID.String(),
		LocationID:  model.LocationID.String(),
		Name:        model.Name,
		Type:        model.Type,
		Price:       helper.Int64FromPg(model.Price),
		Description: model.Description.String,
		CreatedAt:   model.CreatedAt.Time.Format(constant.DateFormat),
		UpdatedAt:   model.UpdatedAt.Time.Format(constant.DateFormat),
	}
}

type GetFieldsResponse struct {
	Fields     []FieldResponse `json:"fields"`
	TotalItems int             `json:"total_items"`
	TotalPages int             `json:"total_pages"`
}

func (f *GetFieldsResponse) FromModel(fields []repository.Field, totalItems, limit int) {
	f.TotalItems = totalItems
	f.TotalPages = helper.CalculateTotalPages(totalItems, limit)

	if len(fields) == 0 {
		f.Fields = []FieldResponse{}

		return
	}

	f.Fields = make([]FieldResponse, len(fields))

	for i, field := range fields {
		f.Fields[i] = FieldResponse{}.FromModel(field)
	}
}
