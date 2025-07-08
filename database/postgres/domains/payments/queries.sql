-- name: InsertPayment :one
INSERT into payments (booking_id, payment_method, payment_status, transaction_id)
values ($1, $2, $3, $4)
returning id;

-- name: GetPaymentsByBookingID :many
SELECT * FROM payments WHERE booking_id = $1
ORDER BY created_at DESC;

-- name: GetPayments :many
SELECT * FROM payments
WHERE ($1::text = '' OR payment_method ILIKE '%' || $1 || '%')
  AND ($2::text = '' OR payment_status ILIKE '%' || $2 || '%')
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountPayments :one
SELECT COUNT(*) FROM payments
WHERE ($1::text = '' OR payment_method ILIKE '%' || $1 || '%')
  AND ($2::text = '' OR payment_status ILIKE '%' || $2 || '%');

-- name: UpdatePaymentStatus :exec
UPDATE payments
SET payment_status = $2,
    payment_method = $3,
    paid_at = $4,
    updated_at = now()
WHERE transaction_id = $1;

-- name: UpdatePaymentStatusByBookingID :exec
UPDATE payments
SET payment_status = $2,
    payment_method = $3,
    paid_at = $4,
    updated_at = now()
WHERE booking_id = $1;
