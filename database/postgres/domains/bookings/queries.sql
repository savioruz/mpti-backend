-- name: InsertBooking :one
INSERT INTO bookings (user_id, field_id, booking_date, start_time, end_time, total_price)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: GetBookingById :one
SELECT * FROM bookings WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: CountOverlaps :one
SELECT COUNT(*) FROM bookings
WHERE field_id = $1
  AND booking_date = $2
  AND status IN ('pending', 'confirmed')
  AND (start_time, end_time) OVERLAPS ($3::time, $4::time)
  AND deleted_at IS NULL;

-- name: CancelBooking :exec
UPDATE bookings
SET status = 'canceled',
    canceled_at = now(),
    canceled_by = $3,
    updated_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
RETURNING id;

-- name: ExpireOldBookings :exec
UPDATE bookings
SET status = 'expired',
    updated_at = now()
WHERE status = 'pending'
  AND expires_at < now()
  AND deleted_at IS NULL
RETURNING id;

-- name: GetBookingsByUserId :many
SELECT * FROM bookings
WHERE user_id = $1
  AND deleted_at IS NULL
    AND ($2::text = '' OR status ILIKE '%' || $2 || '%')
ORDER BY booking_date DESC, start_time DESC
LIMIT $3 OFFSET $4;

-- name: CountBookingsByUserId :one
SELECT COUNT(*) FROM bookings
WHERE user_id = $1
  AND deleted_at IS NULL
    AND ($2::text = '' OR status ILIKE '%' || $2 || '%');

-- name: GetBookedTimeSlots :many
SELECT start_time, end_time
FROM bookings
WHERE field_id = $1
  AND booking_date = $2
  AND status IN ('pending', 'confirmed')
ORDER BY start_time;

-- name: UpdateBookingStatus :exec
UPDATE bookings
SET status = $2,
    updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;
