package dto

import "github.com/google/uuid"

type FieldCreateRequest struct {
	LocationID  uuid.UUID `json:"location_id" validate:"required,uuid"`
	Name        string    `json:"name" validate:"required,min=5,max=255"`
	Type        string    `json:"type" validate:"required,min=5,max=100"`
	Price       int64     `json:"price" validate:"numeric,required,min=5000"`
	Description string    `json:"description" validate:"omitempty"`
}

type FieldUpdateRequest struct {
	LocationID  uuid.UUID `json:"location_id" validate:"omitempty,uuid"`
	Name        string    `json:"name" validate:"omitempty,min=5,max=255"`
	Type        string    `json:"type" validate:"omitempty,min=5,max=100"`
	Price       int64     `json:"price" validate:"omitempty,numeric,min=5000"`
	Description string    `json:"description" validate:"omitempty"`
}
