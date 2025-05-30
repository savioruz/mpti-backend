package gdto

type PaginationRequest struct {
	Page   int    `json:"page" query:"page" validate:"omitempty,numeric,min=1"`
	Limit  int    `json:"limit" query:"limit" validate:"omitempty,numeric,min=1,max=100"`
	Filter string `json:"filter" query:"filter" validate:"omitempty,min=3"`
}
