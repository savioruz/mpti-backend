package dto

import (
	"github.com/savioruz/goth/internal/domains/payments/repository"
	"github.com/savioruz/goth/pkg/constant"
	"github.com/savioruz/goth/pkg/helper"
)

type CreatePaymentInvoiceResponse struct {
	ID         string  `json:"id"`
	OrderID    string  `json:"order_id"`
	Amount     int64   `json:"amount"`
	Status     string  `json:"status"`
	ExpiryDate *string `json:"expiry_date,omitempty"`
	PaymentURL *string `json:"payment_url,omitempty"`
}

type PaymentResponse struct {
	ID            string  `json:"id"`
	BookingID     string  `json:"booking_id"`
	PaymentMethod string  `json:"payment_method"`
	PaymentStatus string  `json:"payment_status"`
	TransactionID string  `json:"transaction_id"`
	PaidAt        *string `json:"paid_at,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

func (p PaymentResponse) FromModel(model repository.Payment) PaymentResponse {
	var paidAt *string

	if model.PaidAt.Valid {
		formattedTime := model.PaidAt.Time.Format(constant.FullDateFormat)
		paidAt = &formattedTime
	}

	return PaymentResponse{
		ID:            model.ID.String(),
		BookingID:     model.BookingID.String(),
		PaymentMethod: model.PaymentMethod,
		PaymentStatus: model.PaymentStatus,
		TransactionID: model.TransactionID,
		PaidAt:        paidAt,
		CreatedAt:     model.CreatedAt.Time.Format(constant.FullDateFormat),
		UpdatedAt:     model.UpdatedAt.Time.Format(constant.FullDateFormat),
	}
}

type PaginatedPaymentResponse struct {
	Payments   []PaymentResponse `json:"payments"`
	TotalItems int               `json:"total_items"`
	TotalPages int               `json:"total_pages"`
}

func (p *PaginatedPaymentResponse) FromModel(payments []repository.Payment, totalItems, limit int) {
	p.TotalItems = totalItems
	p.TotalPages = helper.CalculateTotalPages(totalItems, limit)

	if len(payments) == 0 {
		p.Payments = []PaymentResponse{}

		return
	}

	p.Payments = make([]PaymentResponse, len(payments))

	for i, payment := range payments {
		p.Payments[i] = PaymentResponse{}.FromModel(payment)
	}
}
