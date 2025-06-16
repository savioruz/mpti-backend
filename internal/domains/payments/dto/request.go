package dto

type CreatePaymentRequest struct {
	OrderID    string `json:"order_id" validate:"required,uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	Amount     int64  `json:"amount" validate:"required,numeric,min=10000" example:"10000"`
	PayerEmail string `json:"payer_email" validate:"required,email" example:"mail@example.com"`
}

type PaymentCallbackRequest struct {
	ID                 string  `json:"id"`
	ExternalID         string  `json:"external_id"`
	UserID             string  `json:"user_id"`
	IsHigh             bool    `json:"is_high"`
	PaymentMethod      *string `json:"payment_method,omitempty"`
	Status             string  `json:"status"`
	MerchantName       string  `json:"merchant_name"`
	Amount             int     `json:"amount"`
	BankCode           *string `json:"bank_code,omitempty"`
	PaidAmount         int     `json:"paid_amount"`
	PaidAt             *string `json:"paid_at,omitempty"`
	PayerEmail         *string `json:"payer_email,omitempty"`
	Description        string  `json:"description"`
	Created            string  `json:"created"`
	Updated            string  `json:"updated"`
	Currency           *string `json:"currency,omitempty"`
	PaymentChannel     *string `json:"payment_channel,omitempty"`
	PaymentDestination *string `json:"payment_destination,omitempty"`
	SuccessRedirectURL *string `json:"success_redirect_url,omitempty"`
	FailedRedirectURL  *string `json:"failed_redirect_url,omitempty"`
}
