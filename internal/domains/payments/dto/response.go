package dto

type CreatePaymentResponse struct {
	ID         string `json:"id"`
	OrderID    string `json:"order_id"`
	Amount     int64  `json:"amount"`
	Status     string `json:"status"`
	ExpiryDate string `json:"expiry_date"`
	PaymentURL string `json:"payment_url"`
}
