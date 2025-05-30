package dto

type CreateLocationRequest struct {
	Name        string  `json:"name" validate:"required"`
	Latitude    float64 `json:"latitude" validate:"required,latitude"`
	Longitude   float64 `json:"longitude" validate:"required,longitude"`
	Description string  `json:"description" validate:"omitempty"`
}

type UpdateLocationRequest struct {
	Name        string  `json:"name" validate:"omitempty"`
	Latitude    float64 `json:"latitude" validate:"omitempty,latitude"`
	Longitude   float64 `json:"longitude" validate:"omitempty,longitude"`
	Description string  `json:"description" validate:"omitempty"`
}
